package auth

import (
	"net/http"
	"strings"

	gen "project/internal/gen/api"
)

const bearerPrefix = "Bearer "

// Middleware is the apic "auth" middleware: it requires a valid bearer JWT and
// injects the Principal every protected route reads from req.Auth. Which
// routes are protected is decided in apic.yaml (group use: [auth]), not here.
func Middleware(jwt *JWTManager) func(http.Handler) http.Handler {
	cfg := gen.NewConfig()
	unauthenticated := gen.Error(gen.Unauthenticated, "unauthenticated")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				gen.WriteError(w, cfg, unauthenticated)
				return
			}
			principal, err := jwt.Parse(token)
			if err != nil {
				gen.WriteError(w, cfg, unauthenticated)
				return
			}
			ctx := gen.WithAuth(r.Context(), gen.Principal{ID: principal.PublicUUID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	token := strings.TrimPrefix(header, bearerPrefix)
	return token, token != ""
}
