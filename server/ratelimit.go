package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	times []time.Time
}

type ipRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*rateBucket
	limit     int
	window    time.Duration
	lastSweep time.Time
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: map[string]*rateBucket{},
		limit:   limit,
		window:  window,
	}
}

func (l *ipRateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > l.window {
		l.sweep(cutoff)
		l.lastSweep = now
	}
	b := l.buckets[key]
	if b == nil {
		b = &rateBucket{}
		l.buckets[key] = b
	}
	kept := b.times[:0]
	for _, t := range b.times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	b.times = kept
	if len(b.times) >= l.limit {
		return false
	}
	b.times = append(b.times, now)
	return true
}

// sweep drops buckets whose newest hit is older than cutoff so the map does not grow
// with every client IP ever seen. Caller holds l.mu.
func (l *ipRateLimiter) sweep(cutoff time.Time) {
	for key, b := range l.buckets {
		if len(b.times) == 0 || !b.times[len(b.times)-1].After(cutoff) {
			delete(l.buckets, key)
		}
	}
}

func isAuthAbusePath(path string) bool {
	switch path {
	case "/auth/sign-in", "/auth/sign-up",
		"/auth/oidc/verify-credentials", "/auth/oidc/password-login":
		return true
	default:
		return false
	}
}

// AuthLoginRateLimit limits password / credential endpoints per client IP.
func AuthLoginRateLimit() gin.HandlerFunc {
	limiter := newIPRateLimiter(10, time.Minute)
	return func(c *gin.Context) {
		if !isAuthAbusePath(c.Request.URL.Path) {
			c.Next()
			return
		}
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too_many_requests",
			})
			return
		}
		c.Next()
	}
}
