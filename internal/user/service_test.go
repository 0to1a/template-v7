package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gen "project/internal/gen/api"
)

type fakeRepo struct {
	profiles map[string]Profile
}

func (f *fakeRepo) GetProfileByPublicUUID(_ context.Context, publicUUID string) (Profile, error) {
	profile, ok := f.profiles[publicUUID]
	if !ok {
		return Profile{}, ErrUserNotFound
	}
	return profile, nil
}

func (f *fakeRepo) UpdateDisplayName(_ context.Context, publicUUID, displayName string) (Profile, error) {
	profile, ok := f.profiles[publicUUID]
	if !ok {
		return Profile{}, ErrUserNotFound
	}
	profile.DisplayName = displayName
	f.profiles[publicUUID] = profile
	return profile, nil
}

const (
	testPublicUUID    = "00000000-0000-0000-0000-000000000001"
	unknownPublicUUID = "00000000-0000-0000-0000-000000000099"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	repo := &fakeRepo{profiles: map[string]Profile{
		testPublicUUID: {Email: "admin@localhost", DisplayName: "Admin"},
	}}
	return NewService(repo)
}

func asPrincipal(id string) gen.Principal { return gen.Principal{ID: id} }

func ptr(s string) *string { return &s }

// assertCode fails unless err is a *gen.Status carrying want.
func assertCode(t *testing.T, err error, want gen.Code) {
	t.Helper()
	var st *gen.Status
	if !errors.As(err, &st) || st.Code != want {
		t.Fatalf("err = %v, want status code %s", err, want)
	}
}

func TestGetProfile(t *testing.T) {
	service := newTestService(t)

	profile, err := service.GetProfile(context.Background(), &gen.GetProfileRequest{Auth: asPrincipal(testPublicUUID)})
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Email != "admin@localhost" || profile.DisplayName != "Admin" {
		t.Fatalf("profile = %+v, want email=admin@localhost display_name=Admin", profile)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	service := newTestService(t)

	_, err := service.GetProfile(context.Background(), &gen.GetProfileRequest{Auth: asPrincipal(unknownPublicUUID)})
	assertCode(t, err, gen.NotFound)
}

func TestUpdateProfile_Trims(t *testing.T) {
	service := newTestService(t)

	profile, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(testPublicUUID), DisplayName: ptr("  Ada Lovelace  "),
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if profile.DisplayName != "Ada Lovelace" {
		t.Fatalf("display_name = %q, want %q", profile.DisplayName, "Ada Lovelace")
	}
}

func TestUpdateProfile_EmptyAllowed(t *testing.T) {
	service := newTestService(t)

	profile, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(testPublicUUID), DisplayName: ptr("   "),
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if profile.DisplayName != "" {
		t.Fatalf("display_name = %q, want empty string", profile.DisplayName)
	}
}

func TestUpdateProfile_TooLong(t *testing.T) {
	service := newTestService(t)

	tooLong := "  " + strings.Repeat("a", maxDisplayNameLength+1) + "  "
	_, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(testPublicUUID), DisplayName: ptr(tooLong),
	})
	assertCode(t, err, gen.InvalidArgument)
}

func TestUpdateProfile_AtMaxLength(t *testing.T) {
	service := newTestService(t)

	exact := strings.Repeat("a", maxDisplayNameLength)
	profile, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(testPublicUUID), DisplayName: ptr(exact),
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if profile.DisplayName != exact {
		t.Fatalf("display_name length = %d, want %d", len(profile.DisplayName), len(exact))
	}
}

func TestUpdateProfile_NilClears(t *testing.T) {
	service := newTestService(t)

	profile, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(testPublicUUID), DisplayName: nil,
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if profile.DisplayName != "" {
		t.Fatalf("display_name = %q, want empty string", profile.DisplayName)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	service := newTestService(t)

	_, err := service.UpdateProfile(context.Background(), &gen.UpdateProfileRequest{
		Auth: asPrincipal(unknownPublicUUID), DisplayName: ptr("New Name"),
	})
	assertCode(t, err, gen.NotFound)
}

// fullService forwards only the user methods; gen.Service is embedded nil
// so RegisterRoutes is satisfied without implementing every method.
type fullService struct {
	gen.Service
	svc *Service
}

func (f fullService) GetProfile(ctx context.Context, r *gen.GetProfileRequest) (*gen.Profile, error) {
	return f.svc.GetProfile(ctx, r)
}

func (f fullService) UpdateProfile(ctx context.Context, r *gen.UpdateProfileRequest) (*gen.Profile, error) {
	return f.svc.UpdateProfile(ctx, r)
}

func fixedPrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(gen.WithAuth(r.Context(), gen.Principal{ID: testPublicUUID})))
	})
}

func TestUpdateProfile_HTTPEmptyBodyClearsName(t *testing.T) {
	svc := newTestService(t)
	mux := http.NewServeMux()
	if err := gen.RegisterRoutes(mux, fullService{svc: svc}, gen.Middlewares{Auth: fixedPrincipal}); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"display_name":""}`, `{}`} {
		req := httptest.NewRequest("PUT", "/api/profile", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("body %s: status = %d, want 200; resp = %s", body, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"display_name":""`) {
			t.Fatalf("body %s: resp = %s, want cleared display_name", body, rec.Body.String())
		}
	}
}
