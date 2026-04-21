package handler

import (
	"container/list"
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

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
				} else {
					Error(w, http.StatusUnauthorized, "unauthorized")
					return
				}
			} else if len(header) != 0 {
				w.Header().Add("WWW-Authenticate", "Bearer")
				Error(w, http.StatusUnauthorized, "unauthorized")
				return
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

const rateLimitMaxEntries = 10_000

type lruEntry struct {
	limiter *rate.Limiter
	elem    *list.Element
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*lruEntry
	order   *list.List
	r       rate.Limit
	burst   int
}

func newIPRateLimiter(count int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		entries: make(map[string]*lruEntry, rateLimitMaxEntries),
		order:   list.New(),
		r:       rate.Limit(float64(count) / window.Seconds()),
		burst:   count,
	}
}

func (l *ipRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if e, ok := l.entries[ip]; ok {
		l.order.MoveToFront(e.elem)
		return e.limiter
	}

	if len(l.entries) >= rateLimitMaxEntries {
		back := l.order.Back()
		if back != nil {
			l.order.Remove(back)
			delete(l.entries, back.Value.(string))
		}
	}

	lim := rate.NewLimiter(l.r, l.burst)
	elem := l.order.PushFront(ip)
	l.entries[ip] = &lruEntry{limiter: lim, elem: elem}
	return lim
}

// RateLimit returns a middleware that allows at most count requests per window per IP.
// Per-IP limiters are kept in an LRU capped at rateLimitMaxEntries to bound memory.
func RateLimit(count int, window time.Duration) func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(count, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !limiter.get(ip).Allow() {
				slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				Error(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TODO: Check for TRUSTED_PROXY_IPS before using `X-Real-IP` and `X-Forwarded-For` headers
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
