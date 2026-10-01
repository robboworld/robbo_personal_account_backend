package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func ingest(t *testing.T, authHeader string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/internal/lms/notifications", strings.NewReader(`{}`))
	if authHeader != "" {
		c.Request.Header.Set("Authorization", authHeader)
	}
	NotificationsHandler{}.LMSIngest(c)
	return w.Code
}

func TestLMSIngestRejectsBadTokens(t *testing.T) {
	viper.Set("lmsNotifications.enabled", true)
	t.Cleanup(func() {
		viper.Set("lmsNotifications.enabled", false)
		viper.Set("lmsNotifications.ingestBearerToken", "")
	})
	viper.Set("lmsNotifications.ingestBearerToken", "s3cret-token")
	for _, h := range []string{"", "Bearer ", "Bearer wrong", "Bearer s3cret-tokenX"} {
		if code := ingest(t, h); code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status=%d want 401", h, code)
		}
	}
	// An unset server token must never match an empty / missing client token.
	viper.Set("lmsNotifications.ingestBearerToken", "")
	if code := ingest(t, "Bearer "); code != http.StatusUnauthorized {
		t.Errorf("empty configured token: status=%d want 401", code)
	}
	viper.Set("lmsNotifications.ingestBearerToken", "s3cret-token")
	if code := ingest(t, "Bearer s3cret-token"); code != http.StatusBadRequest {
		t.Errorf("valid token with empty body: status=%d want 400", code)
	}
}
