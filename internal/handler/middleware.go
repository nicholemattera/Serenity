package handler

import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	"github.com/nicholemattera/serenity/internal/service"
)

type contextKey string

const (
	claimsKey contextKey = "claims"
	loggerKey contextKey = "logger"
)

// SlogLogger logs each HTTP request via slog. Must run after RequestID middleware.
// Logs method, path, status, and duration. Never logs headers or query strings
// to avoid accidentally capturing tokens or secrets.
func SlogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		reqID := chimiddleware.GetReqID(r.Context())
		logger := slog.Default().With("request_id", reqID)
		r = r.WithContext(context.WithValue(r.Context(), loggerKey, logger))

		next.ServeHTTP(ww, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// LogFrom returns the request-scoped slog.Logger (with request_id attached),
// falling back to slog.Default() when called outside of SlogLogger.
func LogFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Recoverer recovers from panics, logs the panic and stack trace via slog,
// and writes 500. Re-panics http.ErrAbortHandler to let the server abort cleanly.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}
				LogFrom(r.Context()).Error("panic recovered",
					"panic", fmt.Sprintf("%v", rvr),
					"stack", string(debug.Stack()),
				)
				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func MaxBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

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

// TrustedProxies holds a validated set of trusted proxy IPs and CIDRs.
type TrustedProxies struct {
	ips   map[string]struct{}
	cidrs []*net.IPNet
}

// ParseTrustedProxies parses a slice of IP addresses and CIDR ranges into a TrustedProxies set.
// Returns an error if any entry is not a valid IP or CIDR.
func ParseTrustedProxies(addrs []string) (*TrustedProxies, error) {
	t := &TrustedProxies{ips: make(map[string]struct{})}
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		if strings.ContainsRune(addr, '/') {
			_, cidr, err := net.ParseCIDR(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", addr, err)
			}
			t.cidrs = append(t.cidrs, cidr)
		} else {
			if net.ParseIP(addr) == nil {
				return nil, fmt.Errorf("invalid trusted proxy IP %q", addr)
			}
			t.ips[addr] = struct{}{}
		}
	}
	return t, nil
}

func (t *TrustedProxies) contains(ip string) bool {
	if _, ok := t.ips[ip]; ok {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range t.cidrs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

// RateLimit returns a middleware that allows at most count requests per window per IP.
// Per-IP limiters are kept in an LRU capped at rateLimitMaxEntries to bound memory.
func RateLimit(count int, window time.Duration, proxies *TrustedProxies) func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(count, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r, proxies)
			if !limiter.get(ip).Allow() {
				LogFrom(r.Context()).Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				Error(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request, proxies *TrustedProxies) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if proxies.contains(host) {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			if i := strings.IndexByte(fwd, ','); i >= 0 {
				return strings.TrimSpace(fwd[:i])
			}
			return strings.TrimSpace(fwd)
		}
	}

	return host
}
