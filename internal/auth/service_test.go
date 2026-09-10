package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	gen "project/internal/gen/api"
)

type fakeRepo struct {
	users map[string]User
}

func (f *fakeRepo) GetActiveUserByEmail(_ context.Context, normalizedEmail string) (User, error) {
	user, ok := f.users[normalizedEmail]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

// Mimics the DB unique-email constraint
func (f *fakeRepo) CreateUser(_ context.Context, normalizedEmail string) (User, error) {
	if _, ok := f.users[normalizedEmail]; ok {
		return User{}, errors.New("fakeRepo: user already exists")
	}
	user := User{PublicUUID: "guest-" + normalizedEmail, Email: normalizedEmail}
	f.users[normalizedEmail] = user
	return user, nil
}

type recordingDelivery struct {
	sent []string
}

func (d *recordingDelivery) SendLoginCode(_ context.Context, _, code string) error {
	d.sent = append(d.sent, code)
	return nil
}

const adminUUID = "00000000-0000-0000-0000-000000000001"

func newTestService(t *testing.T, now time.Time) (*Service, *recordingDelivery) {
	t.Helper()
	return newTestServiceWithGuestRegistration(t, now, false)
}

func newTestServiceWithGuestRegistration(t *testing.T, now time.Time, isGuestRegistration bool) (*Service, *recordingDelivery) {
	t.Helper()
	jwtManager, err := NewJWTManager(jwtTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	jwtManager.now = func() time.Time { return now }

	repo := &fakeRepo{users: map[string]User{
		"admin@localhost":  {PublicUUID: adminUUID, Email: "admin@localhost"},
		"user@example.com": {PublicUUID: testUUID, Email: "user@example.com"},
	}}
	delivery := &recordingDelivery{}

	service := NewService(repo, delivery, jwtManager, isGuestRegistration)
	service.now = func() time.Time { return now }
	return service, delivery
}

func isCode(err error, want gen.Code) bool {
	var st *gen.Status
	return errors.As(err, &st) && st.Code == want
}

// TC-001-2: same result for known and unknown emails
func TestRequestLogin_TC001_2(t *testing.T) {
	service, delivery := newTestService(t, fixedTime)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "user@example.com"}); err != nil {
		t.Fatalf("registered email: %v", err)
	}
	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "nobody@example.com"}); err != nil {
		t.Fatalf("unregistered email must get the same generic result, got: %v", err)
	}

	// Response is identical; delivery differs out-of-band
	if len(delivery.sent) != 1 {
		t.Fatalf("expected exactly one code delivered, got %d", len(delivery.sent))
	}
}

// TC-001-3: admin logs in with fixed code, gets JWT
func TestSubmitLogin_TC001_3(t *testing.T) {
	service, _ := newTestService(t, fixedTime)

	resp, err := service.SubmitLogin(context.Background(), &gen.SubmitLoginRequest{Email: "admin@localhost", Code: "123456"})
	if err != nil {
		t.Fatal(err)
	}

	principal, err := service.jwtManager.Parse(resp.AccessToken)
	if err != nil {
		t.Fatalf("issued token failed validation: %v", err)
	}
	if principal.PublicUUID != adminUUID {
		t.Fatalf("subject = %s, want %s", principal.PublicUUID, adminUUID)
	}
}

// TC-001-4: wrong code and unknown user get same error
func TestSubmitLogin_TC001_4(t *testing.T) {
	service, _ := newTestService(t, fixedTime)
	ctx := context.Background()

	resp, err := service.SubmitLogin(ctx, &gen.SubmitLoginRequest{Email: "admin@localhost", Code: "654321"})
	if !isCode(err, gen.Unauthenticated) {
		t.Fatalf("wrong code: err = %v, want gen.Unauthenticated", err)
	}
	if resp != nil {
		t.Fatal("wrong code still produced a token")
	}

	_, unknownErr := service.SubmitLogin(ctx, &gen.SubmitLoginRequest{Email: "nobody@example.com", Code: "123456"})
	if !isCode(unknownErr, gen.Unauthenticated) {
		t.Fatalf("unknown user: err = %v, want the same generic error", unknownErr)
	}
}

// TOTP user: previous step's code must be rejected
func TestSubmitLogin_TOTPUser(t *testing.T) {
	service, delivery := newTestService(t, fixedTime)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "user@example.com"}); err != nil {
		t.Fatal(err)
	}
	if len(delivery.sent) != 1 {
		t.Fatalf("expected one delivered code, got %d", len(delivery.sent))
	}

	if _, err := service.SubmitLogin(ctx, &gen.SubmitLoginRequest{Email: "user@example.com", Code: delivery.sent[0]}); err != nil {
		t.Fatalf("current TOTP rejected: %v", err)
	}

	stale := generateCode([]byte(jwtTestSecret), testUUID, "user@example.com", fixedTime.Add(-totpPeriod))
	if _, err := service.SubmitLogin(ctx, &gen.SubmitLoginRequest{Email: "user@example.com", Code: stale}); !isCode(err, gen.Unauthenticated) {
		t.Fatalf("previous-step TOTP: err = %v, want gen.Unauthenticated", err)
	}
}

// TC-005-1: guest reg creates user and sends code
func TestRequestLogin_TC005_1(t *testing.T) {
	service, delivery := newTestServiceWithGuestRegistration(t, fixedTime, true)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "new@example.com"}); err != nil {
		t.Fatalf("RequestLogin: %v", err)
	}

	if _, err := service.repo.GetActiveUserByEmail(ctx, "new@example.com"); err != nil {
		t.Fatalf("expected a user to now exist for new@example.com: %v", err)
	}
	if len(delivery.sent) != 1 {
		t.Fatalf("expected exactly one code delivered, got %d", len(delivery.sent))
	}
}

// TC-005-2: no guest reg, no user, no code sent
func TestRequestLogin_TC005_2(t *testing.T) {
	service, delivery := newTestService(t, fixedTime)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "new@example.com"}); err != nil {
		t.Fatalf("RequestLogin: %v", err)
	}

	if _, err := service.repo.GetActiveUserByEmail(ctx, "new@example.com"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected no user to be created, got err = %v", err)
	}
	if len(delivery.sent) != 0 {
		t.Fatalf("expected no code delivered, got %d", len(delivery.sent))
	}
}

// TC-005-3: guest reg dedupes an existing user
func TestRequestLogin_TC005_3(t *testing.T) {
	service, delivery := newTestServiceWithGuestRegistration(t, fixedTime, true)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "admin@localhost"}); err != nil {
		t.Fatalf("RequestLogin: %v", err)
	}

	user, err := service.repo.GetActiveUserByEmail(ctx, "admin@localhost")
	if err != nil {
		t.Fatalf("existing user vanished: %v", err)
	}
	if user.PublicUUID != adminUUID {
		t.Fatalf("existing user was replaced: PublicUUID = %s, want %s", user.PublicUUID, adminUUID)
	}
	if len(delivery.sent) != 1 {
		t.Fatalf("expected exactly one code delivered, got %d", len(delivery.sent))
	}
}

// TC-005-4: auto-registered guest completes login
func TestSubmitLogin_TC005_4(t *testing.T) {
	service, delivery := newTestServiceWithGuestRegistration(t, fixedTime, true)
	ctx := context.Background()

	if err := service.RequestLogin(ctx, &gen.RequestLoginRequest{Email: "new@example.com"}); err != nil {
		t.Fatalf("RequestLogin: %v", err)
	}
	if len(delivery.sent) != 1 {
		t.Fatalf("expected exactly one code delivered, got %d", len(delivery.sent))
	}

	resp, err := service.SubmitLogin(ctx, &gen.SubmitLoginRequest{Email: "new@example.com", Code: delivery.sent[0]})
	if err != nil {
		t.Fatalf("SubmitLogin: %v", err)
	}
	if _, err := service.jwtManager.Parse(resp.AccessToken); err != nil {
		t.Fatalf("issued token failed validation: %v", err)
	}
}
