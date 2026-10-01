package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// A proxy-supplied id is kept only if it is short and plain, so it cannot inject log lines.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// requestID tags each request with X-Request-ID (taken from the proxy or generated) in the
// response and in the access log, so a user report can be matched to log lines.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("t%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// accessLog is gin's request log without the query string: play links carry ?token=... and
// OIDC callbacks carry codes, which must not end up in logs.
func accessLog() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		id, _ := p.Keys["request_id"].(string)
		return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s | rid=%s\n",
			p.TimeStamp.Format("2006/01/02 - 15:04:05"),
			p.StatusCode, p.Latency, p.ClientIP, p.Method, p.Request.URL.Path, id)
	})
}

// ginMode runs gin in release mode unless GIN_MODE says otherwise: debug mode prints every
// route at startup and warns on each request.
func ginMode() {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
}
