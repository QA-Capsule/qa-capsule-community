package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

var globalRateLimiter = &rateLimiter{hits: make(map[string][]time.Time)}

func (rl *rateLimiter) allow(key string, max int, window time.Duration) bool {
	if max <= 0 || window <= 0 {
		return true
	}
	now := time.Now()
	cutoff := now.Add(-window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := rl.hits[key]
	filtered := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= max {
		rl.hits[key] = filtered
		return false
	}
	filtered = append(filtered, now)
	rl.hits[key] = filtered
	return true
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func rateLimitMiddleware(bucket string, max int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := bucket + ":" + clientIP(r)
		if !globalRateLimiter.allow(key, max, window) {
			writeJSONError(w, "Too many requests — try again later", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
