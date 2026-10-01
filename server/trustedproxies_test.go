package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func clientIPVia(t *testing.T, trusted, remoteAddr, xff string) string {
	t.Helper()
	t.Setenv("TRUSTED_PROXIES", trusted)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := applyTrustedProxies(engine); err != nil {
		t.Fatalf("applyTrustedProxies: %v", err)
	}
	engine.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Body.String()
}

func TestClientIPIgnoresSpoofedForwardingWithoutTrustedProxies(t *testing.T) {
	if got := clientIPVia(t, "", "203.0.113.5:4242", "185.71.76.1"); got != "203.0.113.5" {
		t.Fatalf("ClientIP=%q want TCP peer 203.0.113.5", got)
	}
}

func TestClientIPUsesForwardingFromTrustedProxy(t *testing.T) {
	if got := clientIPVia(t, "10.0.0.0/8", "10.0.0.2:4242", "198.51.100.7"); got != "198.51.100.7" {
		t.Fatalf("ClientIP=%q want forwarded 198.51.100.7", got)
	}
}

func TestClientIPSkipsSpoofedHeadEntryBehindTrustedProxy(t *testing.T) {
	// Client sends "X-Forwarded-For: 185.71.76.1"; the proxy appends the real peer.
	got := clientIPVia(t, "10.0.0.0/8", "10.0.0.2:4242", "185.71.76.1, 198.51.100.7")
	if got != "198.51.100.7" {
		t.Fatalf("ClientIP=%q want right-most untrusted 198.51.100.7", got)
	}
}

func TestClientIPUntrustedPeerCannotForward(t *testing.T) {
	if got := clientIPVia(t, "10.0.0.0/8", "203.0.113.5:4242", "198.51.100.7"); got != "203.0.113.5" {
		t.Fatalf("ClientIP=%q want TCP peer 203.0.113.5", got)
	}
}

func TestRateLimiterSweepsStaleBuckets(t *testing.T) {
	l := newIPRateLimiter(5, 10*time.Millisecond)
	l.allow("198.51.100.1")
	l.allow("198.51.100.2")
	time.Sleep(25 * time.Millisecond)
	l.allow("198.51.100.3")
	if _, ok := l.buckets["198.51.100.1"]; ok {
		t.Fatal("stale bucket was not swept")
	}
	if len(l.buckets) != 1 {
		t.Fatalf("buckets=%d want 1", len(l.buckets))
	}
}
