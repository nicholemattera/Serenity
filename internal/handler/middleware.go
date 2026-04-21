package handler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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

// RateLimit returns a middleware that allows at most rate requests per window per IP.
func RateLimit(rate int, window time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		mu      sync.Mutex
		count   int
		resetAt time.Time
	}
	var m sync.Map

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			v, _ := m.LoadOrStore(ip, &entry{})
			e := v.(*entry)

			e.mu.Lock()
			now := time.Now()
			if now.After(e.resetAt) {
				e.count = 0
				e.resetAt = now.Add(window)
			}
			e.count++
			count := e.count
			e.mu.Unlock()

			if count > rate {
				slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				Error(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i >= 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
