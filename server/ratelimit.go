package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	times []time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
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

func requestIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return c.Request.RemoteAddr
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
		if !limiter.allow(requestIP(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too_many_requests",
			})
			return
		}
		c.Next()
	}
}
