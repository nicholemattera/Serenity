package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicholemattera/serenity/internal/handler"
)

func rateLimitedOKHandler(count int, window time.Duration, proxies *handler.TrustedProxies) http.Handler {
	return handler.RateLimit(count, window, proxies)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func doRateLimited(h http.Handler, remoteAddr, xRealIP, xForwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = remoteAddr
	if xRealIP != "" {
		req.Header.Set("X-Real-IP", xRealIP)
	}
	if xForwardedFor != "" {
		req.Header.Set("X-Forwarded-For", xForwardedFor)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestParseTrustedProxies(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty strings ignored", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies([]string{"", ""}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid IP", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies([]string{"192.168.1.1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid CIDR", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid IP", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies([]string{"not-an-ip"}); err == nil {
			t.Fatal("expected error for invalid IP")
		}
	})

	t.Run("invalid CIDR", func(t *testing.T) {
		if _, err := handler.ParseTrustedProxies([]string{"10.0.0.0/99"}); err == nil {
			t.Fatal("expected error for invalid CIDR")
		}
	})
}

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies(nil)
	h := rateLimitedOKHandler(3, time.Minute, proxies)

	for i := range 3 {
		rr := doRateLimited(h, "1.2.3.4:1000", "", "")
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimit_BlocksOverBurst(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies(nil)
	h := rateLimitedOKHandler(2, time.Minute, proxies)

	doRateLimited(h, "1.2.3.4:1000", "", "")
	doRateLimited(h, "1.2.3.4:1000", "", "")

	rr := doRateLimited(h, "1.2.3.4:1000", "", "")
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestRateLimit_TracksIPsIndependently(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies(nil)
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	// Exhaust IP A.
	doRateLimited(h, "1.1.1.1:1000", "", "")
	rr := doRateLimited(h, "1.1.1.1:1000", "", "")
	assertStatus(t, rr, http.StatusTooManyRequests)

	// IP B is unaffected.
	rr = doRateLimited(h, "2.2.2.2:1000", "", "")
	assertStatus(t, rr, http.StatusOK)
}

func TestRateLimit_TrustedProxy_UsesXRealIP(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies([]string{"10.0.0.1"})
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	// First request from real client IP via trusted proxy — allowed.
	rr := doRateLimited(h, "10.0.0.1:1000", "5.5.5.5", "")
	assertStatus(t, rr, http.StatusOK)

	// Second request from same real client IP — blocked.
	rr = doRateLimited(h, "10.0.0.1:1000", "5.5.5.5", "")
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestRateLimit_TrustedProxy_UsesXForwardedFor(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies([]string{"10.0.0.1"})
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	rr := doRateLimited(h, "10.0.0.1:1000", "", "5.5.5.5, 10.0.0.1")
	assertStatus(t, rr, http.StatusOK)

	rr = doRateLimited(h, "10.0.0.1:1000", "", "5.5.5.5, 10.0.0.1")
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestRateLimit_UntrustedProxy_IgnoresForwardedHeaders(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies([]string{"10.0.0.1"})
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	// Untrusted remote addr with spoofed headers — rate key is the remote addr.
	doRateLimited(h, "9.9.9.9:1000", "5.5.5.5", "")

	// Same remote addr, different spoofed X-Real-IP — still limited.
	rr := doRateLimited(h, "9.9.9.9:1000", "6.6.6.6", "")
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestRateLimit_TrustedCIDR(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies([]string{"172.16.0.0/12"})
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	// IP within the trusted CIDR range.
	rr := doRateLimited(h, "172.16.0.5:1000", "5.5.5.5", "")
	assertStatus(t, rr, http.StatusOK)

	rr = doRateLimited(h, "172.16.0.5:1000", "5.5.5.5", "")
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestRateLimit_TrustedCIDR_OutsideRange(t *testing.T) {
	proxies, _ := handler.ParseTrustedProxies([]string{"172.16.0.0/12"})
	h := rateLimitedOKHandler(1, time.Minute, proxies)

	// IP outside the CIDR — forwarded headers ignored, remote addr used.
	doRateLimited(h, "192.168.1.1:1000", "5.5.5.5", "")

	// Same remote addr, different header — still limited by remote addr.
	rr := doRateLimited(h, "192.168.1.1:1000", "6.6.6.6", "")
	assertStatus(t, rr, http.StatusTooManyRequests)
}
