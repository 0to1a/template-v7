package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gen "project/internal/gen/api"
)

// probeService records the Principal that reached GetProfile. Embedding the
// nil gen.Service satisfies the interface; only GetProfile is ever called.
type probeService struct {
	gen.Service
	got *gen.Principal
}

func (p probeService) GetProfile(_ context.Context, req *gen.GetProfileRequest) (*gen.Profile, error) {
	*p.got = req.Auth
	return &gen.Profile{}, nil
}

func newProbe(t *testing.T, m *JWTManager) (http.Handler, *gen.Principal) {
	t.Helper()
	got := &gen.Principal{}
	mux := http.NewServeMux()
	if err := gen.RegisterRoutes(mux, probeService{got: got}, gen.Middlewares{Auth: Middleware(m)}); err != nil {
		t.Fatal(err)
	}
	return mux, got
}

func TestMiddleware_MissingHeaderRejected(t *testing.T) {
	h, got := newProbe(t, newTestJWTManager(t, fixedTime))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/profile", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"UNAUTHENTICATED"`) {
		t.Fatalf("body = %s, want UNAUTHENTICATED envelope", rec.Body.String())
	}
	if got.ID != "" {
		t.Fatal("handler reached without authentication")
	}
}

func TestMiddleware_MalformedTokenRejected(t *testing.T) {
	h, got := newProbe(t, newTestJWTManager(t, fixedTime))

	req := httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got.ID != "" {
		t.Fatal("handler reached with a malformed token")
	}
}

func TestMiddleware_ValidTokenInjectsPrincipal(t *testing.T) {
	m := newTestJWTManager(t, fixedTime)
	token, err := m.Issue(testUUID)
	if err != nil {
		t.Fatal(err)
	}
	h, got := newProbe(t, m)

	req := httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got.ID != testUUID {
		t.Fatalf("principal id = %q, want %q", got.ID, testUUID)
	}
}
