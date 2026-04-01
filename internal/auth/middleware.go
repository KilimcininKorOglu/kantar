package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsKey contextKey = "claims"

// Middleware extracts and validates the JWT token from the Authorization header.
// On success, the Claims are stored in the request context.
func Middleware(jwtMgr *JWTManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtMgr.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves the Claims from the request context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

func extractToken(r *http.Request) string {
	// Bearer token from Authorization header (CLI/API clients)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Fallback: HttpOnly cookie (browser requests)
	if c, err := r.Cookie("kantar_token"); err == nil && c.Value != "" {
		return c.Value
	}

	return ""
}
