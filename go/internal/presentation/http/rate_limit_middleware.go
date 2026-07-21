package http

import (
	"log/slog"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// rateLimiter tracks one token-bucket limiter per client IP. A single
// shared instance is wrapped around whichever routes need protection —
// each route can use a different (rps, burst) budget via withRateLimit.
type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newRateLimiter(requestsPerSecond float64, burst int) *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

func (rl *rateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, ok := rl.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[ip] = limiter
	}
	return limiter
}

// withRateLimit rejects requests once a client IP exceeds its budget,
// logging the offending IP and path so repeated abuse is visible.
func withRateLimit(rl *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.limiterFor(ip).Allow() {
			slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the real client address. X-Real-IP is only trusted
// because our nginx config overwrites it with $remote_addr on every
// request (proxy_set_header X-Real-IP $remote_addr) — nginx replaces
// any client-supplied value rather than appending to it, so a client
// cannot spoof this as long as pgscope is only reachable through that
// nginx, never directly. Falls back to RemoteAddr for local/no-proxy use.
func clientIP(r *http.Request) string {
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return real
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
