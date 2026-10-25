package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func passwordRoutes(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	Handler{}.InitRoutes(engine)
	return engine
}

func postJSON(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	return w
}

func setPasswordFallback(t *testing.T, mode string, fallback bool) {
	t.Helper()
	viper.Set("auth.mode", mode)
	viper.Set("auth.lmsPasswordFallback", fallback)
	t.Cleanup(func() {
		viper.Set("auth.mode", "")
		viper.Set("auth.lmsPasswordFallback", false)
	})
}

func TestPasswordEndpointsClosedWithoutFallback(t *testing.T) {
	setPasswordFallback(t, "oidc_bff", false)
	engine := passwordRoutes(t)
	for _, path := range []string{"/auth/oidc/password-login", "/auth/oidc/verify-credentials"} {
		if w := postJSON(engine, path); w.Code != http.StatusForbidden {
			t.Fatalf("%s: status=%d want 403 (body %q)", path, w.Code, w.Body.String())
		}
	}
}

// With the fallback on the guard steps aside; the malformed body then fails validation.
func TestPasswordEndpointsOpenWithFallback(t *testing.T) {
	for _, tc := range []struct {
		mode     string
		fallback bool
	}{{"oidc_bff", true}, {"lms_db", false}} {
		setPasswordFallback(t, tc.mode, tc.fallback)
		engine := passwordRoutes(t)
		for _, path := range []string{"/auth/oidc/password-login", "/auth/oidc/verify-credentials"} {
			if w := postJSON(engine, path); w.Code != http.StatusBadRequest {
				t.Fatalf("mode=%s fallback=%v %s: status=%d want 400", tc.mode, tc.fallback, path, w.Code)
			}
		}
	}
}
