package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/amsoft/servicewatch/internal/auth"
)

type contextKey string

const userClaimsKey contextKey = "user_claims"

// Authenticate extracts and validates the Bearer token from the Authorization
// header. It sets the claims in context and calls next on success.
func Authenticate(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r)
			if !ok {
				WriteError(w, http.StatusUnauthorized, "missing or malformed authorization header")
				return
			}

			claims, err := authSvc.ValidateToken(token)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves JWT claims stored by the Authenticate middleware.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(userClaimsKey).(*auth.Claims)
	return c, ok
}

func extractBearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
