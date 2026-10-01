package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccessLogOmitsQueryAndTagsRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	gin.DefaultWriter = &buf
	t.Cleanup(func() { gin.DefaultWriter = nil })
	engine := gin.New()
	engine.Use(requestID(), accessLog())
	engine.GET("/projectPage/:id/sb3", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/projectPage/1/sb3?token=secret-play-token", nil)
	req.Header.Set(requestIDHeader, "abc-123")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	line := buf.String()
	if strings.Contains(line, "secret-play-token") {
		t.Fatalf("query string logged: %s", line)
	}
	if !strings.Contains(line, "/projectPage/1/sb3") || !strings.Contains(line, "rid=abc-123") {
		t.Fatalf("unexpected log line: %s", line)
	}
	if w.Header().Get(requestIDHeader) != "abc-123" {
		t.Fatalf("response request id=%q", w.Header().Get(requestIDHeader))
	}
}

func TestRequestIDRejectsUnsafeIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(requestID())
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, "x\nFAKE LOG LINE")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if got := w.Header().Get(requestIDHeader); got == "" || strings.ContainsAny(got, "\n ") {
		t.Fatalf("request id=%q", got)
	}
}
