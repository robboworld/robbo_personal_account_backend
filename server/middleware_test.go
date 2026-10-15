package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestTokenAuthRejectsMalformedHeaderWithoutPanic(t *testing.T) {
	viper.Set("auth.mode", "")
	viper.Set("oidc.enabled", false)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(TokenAuthMiddleware(fakeSessions{}))
	engine.POST("/query", func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, h := range []string{"Bearer", "Bearer a b", "garbage"} {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", h)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Authorization=%q: status=%d want 401", h, w.Code)
		}
	}
}

// The RS3 device routes carry the licence's offline token as Bearer: the middleware must leave it
// to the licensing handlers instead of checking it as a user access token.
func TestTokenAuthPassesDeviceTokenRoutes(t *testing.T) {
	viper.Set("auth.mode", "")
	viper.Set("oidc.enabled", false)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(TokenAuthMiddleware(fakeSessions{}))
	paths := []string{"/v1/token/refresh", "/v1/seats/release"}
	for _, p := range paths {
		engine.POST(p, func(c *gin.Context) { c.Status(http.StatusOK) })
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p, nil)
		req.Header.Set("Authorization", "Bearer device.offline.token")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status=%d want 200", p, w.Code)
		}
	}
}
