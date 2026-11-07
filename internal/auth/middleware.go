package auth

import (
	"net/http"
	"strings"

	gen "project/internal/gen/api"
)

const bearerPrefix = "Bearer "

// The apic "auth" middleware; which routes use it is decided in apic.yaml
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
			publicUUID, err := jwt.Parse(token)
			if err != nil {
				gen.WriteError(w, cfg, unauthenticated)
				return
			}
			ctx := gen.WithAuth(r.Context(), gen.Principal{ID: publicUUID})
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
