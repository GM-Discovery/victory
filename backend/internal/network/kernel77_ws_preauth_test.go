package network

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/identity"
)

// Every test in this file talks to wsConnectionLimiter, a package-level,
// shared-state limiter (Kernel 77 K77-06). httptest clients all originate
// from 127.0.0.1, so without this, tests would silently share and exhaust
// each other's token bucket depending on run order. Each test claims its
// own fake source IP via X-Forwarded-For (which ratelimit.ClientIP prefers
// over RemoteAddr) to stay isolated.
func getWithFakeIP(t *testing.T, url, fakeIP string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("X-Forwarded-For", fakeIP)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// TestAnonymousWebSocketRejectedBeforeUpgrade proves Kernel 77 K77-08
// (K76-L01): an anonymous or unauthorized caller is refused with a plain
// HTTP status before the handshake completes, not after. A completed
// upgrade followed by a WS error frame (the pre-Kernel-77 behavior) still
// costs a full handshake even when the caller was never going to be let in;
// this asserts the connection is refused at the HTTP layer instead.
func TestAnonymousWebSocketRejectedBeforeUpgrade(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	hub := NewHub()

	handler := ServeVenueWS(hub, pool, identity.DiscordServerLinkConfig{}, "the-cave")
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// A plain (non-websocket) HTTP client hitting this endpoint anonymously
	// must get an ordinary HTTP error response -- if the handshake had
	// already completed, this request would instead hang or return a
	// protocol-mismatch error rather than a clean 401.
	resp := getWithFakeIP(t, server.URL, "198.51.100.1")
	defer resp.Body.Close()

	// An empty session cookie resolves to no user; UserCanAccessVenueSlug
	// then correctly reports that no-user as not allowed, so this surfaces
	// as 403 rather than 401 -- either is a fine outcome for this test:
	// what matters is that it's a plain HTTP status, not a completed
	// upgrade followed by a WS error frame.
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 401 or 403 before any upgrade attempt, got %d", resp.StatusCode)
	}
	if upgradeHeader := resp.Header.Get("Upgrade"); upgradeHeader != "" {
		t.Fatalf("response should not carry an Upgrade header once rejected pre-handshake, got %q", upgradeHeader)
	}
}

// TestWebSocketConnectionAttemptsAreRateLimited proves Kernel 77 K77-06:
// repeated connection attempts from one client eventually hit 429, where
// previously there was no limit at all on connection attempts.
func TestWebSocketConnectionAttemptsAreRateLimited(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	hub := NewHub()

	handler := ServeVenueWS(hub, pool, identity.DiscordServerLinkConfig{}, "the-cave")
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	var lastCode int
	for i := 0; i < 40; i++ {
		resp := getWithFakeIP(t, server.URL, "198.51.100.2")
		lastCode = resp.StatusCode
		resp.Body.Close()
		if lastCode == http.StatusTooManyRequests {
			break
		}
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected the connection limiter to eventually return 429, last code was %d", lastCode)
	}
}

func TestAnonymousProfileWebSocketRejectedBeforeUpgrade(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	hub := NewHub()

	handler := ServeProfileWS(hub, pool)
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	resp := getWithFakeIP(t, server.URL, "198.51.100.3")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 before any upgrade attempt, got %d", resp.StatusCode)
	}
}
