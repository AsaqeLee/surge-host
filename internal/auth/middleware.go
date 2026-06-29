package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/asaqe/surge-host/pkg/response"
)

type contextKey string

const usernameKey contextKey = "username"

// UsernameFromContext returns the authenticated username from request context.
func UsernameFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(usernameKey).(string)
	return u, ok
}

// RequireAuth protects routes with JWT Bearer or Basic Auth.
// When auth is disabled (no password set), requests are allowed as admin user.
func RequireAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !svc.AuthEnabled() {
				ctx := context.WithValue(r.Context(), usernameKey, svc.AdminUsername())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if u, p, ok := r.BasicAuth(); ok && svc.ValidateBasic(u, p) {
				ctx := context.WithValue(r.Context(), usernameKey, u)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token := strings.TrimPrefix(header, "Bearer ")
				claims, err := svc.ValidateToken(token)
				if err == nil {
					ctx := context.WithValue(r.Context(), usernameKey, claims.Username)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Bearer realm="surge-host"`)
			response.Error(w, http.StatusUnauthorized, "authentication required")
		})
	}
}