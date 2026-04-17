package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/nicholemattera/serenity/internal/service"
)

type contextKey string

const claimsKey contextKey = "claims"

// Authenticate is a soft middleware: it extracts and validates the Bearer token
// if present and attaches the claims to the context. It does NOT reject requests
// without a token — handlers decide whether auth is required.
func Authenticate(authSvc service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token := strings.TrimPrefix(header, "Bearer ")
				if claims, err := authSvc.ValidateToken(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth rejects the request with 401 if no valid claims are in the context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetClaims(r) == nil {
			Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetClaims returns the JWT claims from the request context, or nil if unauthenticated.
func GetClaims(r *http.Request) *service.Claims {
	claims, _ := r.Context().Value(claimsKey).(*service.Claims)
	return claims
}
