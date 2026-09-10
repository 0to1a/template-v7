package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtTestSecret = "unit-test-jwt-secret-0123456789abcdef"

func newTestJWTManager(t *testing.T, now time.Time) *JWTManager {
	t.Helper()
	m, err := NewJWTManager(jwtTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return now }
	return m
}

func TestJWT_SecretTooShortRejected(t *testing.T) {
	if _, err := NewJWTManager("short"); err == nil {
		t.Fatal("short JWT secret was accepted")
	}
}

func TestJWT_IssueAndParse(t *testing.T) {
	m := newTestJWTManager(t, fixedTime)

	token, err := m.Issue(testUUID)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := m.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.PublicUUID != testUUID {
		t.Fatalf("subject = %s, want %s", principal.PublicUUID, testUUID)
	}
}

func TestJWT_ExpiredRejected(t *testing.T) {
	issuer := newTestJWTManager(t, fixedTime)
	token, err := issuer.Issue(testUUID)
	if err != nil {
		t.Fatal(err)
	}

	verifier := newTestJWTManager(t, fixedTime.Add(accessTokenTTL+time.Minute))
	if _, err := verifier.Parse(token); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestJWT_NonHS256Rejected(t *testing.T) {
	m := newTestJWTManager(t, fixedTime)

	claims := jwt.RegisteredClaims{
		Subject:   testUUID,
		ExpiresAt: jwt.NewNumericDate(fixedTime.Add(time.Hour)),
	}

	// Rejected by HS256-only allowlist, not sig check
	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(jwtTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Parse(hs512); err == nil {
		t.Fatal("HS512 token was accepted")
	}

	// alg=none must never be accepted.
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Parse(none); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}

func TestJWT_MissingExpirationRejected(t *testing.T) {
	m := newTestJWTManager(t, fixedTime)

	// exp is mandatory, not just validated when present
	claims := jwt.RegisteredClaims{
		Subject: testUUID,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Parse(token); err == nil {
		t.Fatal("token without exp was accepted")
	}
}

func TestJWT_EmptySubjectRejected(t *testing.T) {
	m := newTestJWTManager(t, fixedTime)

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(fixedTime.Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Parse(token); err == nil {
		t.Fatal("token without subject was accepted")
	}
}
