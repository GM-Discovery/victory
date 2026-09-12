// Package ratelimit is a small in-memory, per-client-IP token bucket for the
// credential endpoints (Kernel 72, track O4). Single-instance by design —
// Victory runs one backend container; state resets on restart, which is fine
// for brute-force throttling. It is deliberately NOT applied to general API
// routes: only endpoints that accept a credential guess.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastFill time.Time
}

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens added per second
	burst    float64 // bucket capacity
	lastGC   time.Time
	now      func() time.Time // test seam
	maxIdle  time.Duration
	gcPeriod time.Duration
}

// New returns a limiter allowing `burst` immediate attempts per client and a
// sustained refill of `perMinute` attempts each minute.
func New(burst int, perMinute float64) *Limiter {
	return &Limiter{
		buckets:  map[string]*bucket{},
		rate:     perMinute / 60.0,
		burst:    float64(burst),
		now:      time.Now,
		maxIdle:  30 * time.Minute,
		gcPeriod: 5 * time.Minute,
		lastGC:   time.Now(),
	}
}

// Allow consumes one token for key and reports whether the attempt may
// proceed.
func (l *Limiter) Allow(key string) bool {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastGC) > l.gcPeriod {
		for k, b := range l.buckets {
			if now.Sub(b.lastFill) > l.maxIdle {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastFill: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.lastFill).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.lastFill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ClientIP resolves the caller's address. The backend is only reachable
// through the bread-caddy reverse proxy on the internal Docker network, so
// the first X-Forwarded-For hop is trustworthy; direct connections (local
// dev, tests) fall back to RemoteAddr.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	// Kernel 96: only honor X-Forwarded-For when the connection actually
	// came from a trusted reverse proxy, not unconditionally -- trusting
	// it from anyone meant a caller reaching the backend directly could
	// spoof a fresh value per request and bypass every IP-based rate
	// limit (login throttling included) entirely. "Trusted" means a
	// private-range address rather than strictly loopback: Caddy reaches
	// the backend over a published loopback port in the self-hosted
	// Podman path, but production's bread-caddy reaches it over the
	// edge_net container bridge network instead, which is a private
	// (non-loopback) address, not a public one. The backend is never
	// publicly reachable directly in any current deployment (confirmed
	// elsewhere this pass), so a genuine internet attacker's own
	// RemoteAddr can never be private-range -- this distinction holds
	// regardless of which topology actually put the proxy in front.
	if !isPrivateNetwork(host) {
		return host
	}

	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		first := strings.TrimSpace(strings.Split(fwd, ",")[0])
		if first != "" {
			return first
		}
	}
	return host
}

func isPrivateNetwork(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

// Middleware wraps a credential endpoint with a typed 429 refusal.
func Middleware(l *Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(ClientIP(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"data":{"error":"rate_limited","message":"Too many attempts. Wait a minute and try again."}}`))
			return
		}
		next(w, r)
	}
}
