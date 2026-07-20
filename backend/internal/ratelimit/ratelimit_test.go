package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBurstThenRefusalThenRefill(t *testing.T) {
	current := time.Unix(1000, 0)
	l := New(3, 60) // 3 burst, 1/second refill
	l.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("attempt %d within burst should pass", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("attempt beyond burst should be refused")
	}
	// A different client is unaffected.
	if !l.Allow("b") {
		t.Fatal("second client should have its own bucket")
	}
	// One second later a token has refilled.
	current = current.Add(time.Second)
	if !l.Allow("a") {
		t.Fatal("refilled token should pass")
	}
	if l.Allow("a") {
		t.Fatal("bucket should be empty again")
	}
}

func TestClientIPPrefersForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.RemoteAddr = "172.18.0.5:41234"
	if got := ClientIP(r); got != "172.18.0.5" {
		t.Fatalf("RemoteAddr fallback: got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 172.18.0.2")
	if got := ClientIP(r); got != "203.0.113.9" {
		t.Fatalf("X-Forwarded-For first hop: got %q", got)
	}
}

func TestMiddlewareReturnsTyped429(t *testing.T) {
	l := New(1, 1)
	handler := Middleware(l, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.RemoteAddr = "198.51.100.7:1000"

	first := httptest.NewRecorder()
	handler(first, r)
	if first.Code != http.StatusOK {
		t.Fatalf("first attempt: got %d", first.Code)
	}
	second := httptest.NewRecorder()
	handler(second, r)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second attempt: got %d", second.Code)
	}
	if body := second.Body.String(); !strings.Contains(body, "rate_limited") {
		t.Fatalf("expected typed rate_limited error, got %s", body)
	}
}
