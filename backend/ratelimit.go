package main

import (
	"net/http"
	"sync"
	"time"
)

// loginRateLimiter is a simple per-IP sliding-window rate limiter for /api/auth/login.
type loginRateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	l := &loginRateLimiter{buckets: map[string][]time.Time{}}
	go l.reap() // background cleanup
	return l
}

func (l *loginRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window := now.Add(-1 * time.Minute)

	// Filter to last minute.
	var recent []time.Time
	for _, t := range l.buckets[ip] {
		if t.After(window) {
			recent = append(recent, t)
		}
	}
	l.buckets[ip] = recent

	if len(recent) >= 5 {
		return false
	}
	l.buckets[ip] = append(l.buckets[ip], now)
	return true
}

func (l *loginRateLimiter) reap() {
	for {
		time.Sleep(5 * time.Minute)
		l.mu.Lock()
		cutoff := time.Now().Add(-2 * time.Minute)
		for ip, times := range l.buckets {
			var keep []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					keep = append(keep, t)
				}
			}
			if len(keep) == 0 {
				delete(l.buckets, ip)
			} else {
				l.buckets[ip] = keep
			}
		}
		l.mu.Unlock()
	}
}

// rateLimitLogin wraps a handler with login rate limiting (5 attempts/min per IP).
func rateLimitLogin(limiter *loginRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !limiter.allow(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "too many login attempts — try again in a minute",
			})
			return
		}
		next(w, r)
	}
}
