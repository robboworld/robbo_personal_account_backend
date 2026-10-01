package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func runLogout(t *testing.T, target string, handle func(Handler, *gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	handle(Handler{}, c)
	return w
}

func setLogoutConfig(t *testing.T) {
	t.Helper()
	viper.Set("oidc.logoutEndpoint", "http://lms.test/logout")
	viper.Set("oidc.frontendBaseUrl", "http://lk.test")
	t.Setenv("LMS_URL", "http://lms.test")
	t.Setenv("ROBBO_RS_URL", "http://rs.test")
	t.Cleanup(func() {
		viper.Set("oidc.logoutEndpoint", "")
		viper.Set("oidc.frontendBaseUrl", "")
	})
}

// idpRedirect asserts a top-level 302 to IdP /logout and returns the decoded redirect_url.
func idpRedirect(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != http.StatusFound {
		t.Fatalf("status=%d want 302 (body %q)", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad Location: %v", err)
	}
	if loc.Host != "lms.test" || loc.Path != "/logout" {
		t.Fatalf("Location=%s want IdP /logout", loc)
	}
	q := loc.Query()
	if _, ok := q["amp;redirect_url"]; ok {
		t.Fatalf("HTML-escaped query in Location: %s", loc)
	}
	return q.Get("redirect_url")
}

func TestLogoutFromLKRedirectsThroughIdPToLK(t *testing.T) {
	setLogoutConfig(t)
	w := runLogout(t, "/auth/oidc/logout/lk", Handler.LogoutFromLK)
	if got := idpRedirect(t, w); got != "http://lk.test/" {
		t.Fatalf("redirect_url=%q want http://lk.test/", got)
	}
}

func TestLogoutFromRSKeepsAllowedAbsoluteReturnTo(t *testing.T) {
	setLogoutConfig(t)
	w := runLogout(t, "/auth/oidc/logout/rs?return_to="+url.QueryEscape("http://rs.test/editor"), Handler.LogoutFromRS)
	if got := idpRedirect(t, w); got != "http://rs.test/editor" {
		t.Fatalf("redirect_url=%q want http://rs.test/editor", got)
	}
}

func TestLogoutFromRSFallsBackToRSLanding(t *testing.T) {
	setLogoutConfig(t)
	for _, rt := range []string{"", "https://evil.test/steal", "/home", "//evil.test"} {
		w := runLogout(t, "/auth/oidc/logout/rs?return_to="+url.QueryEscape(rt), Handler.LogoutFromRS)
		if got := idpRedirect(t, w); got != "http://rs.test/" {
			t.Fatalf("return_to=%q: redirect_url=%q want http://rs.test/", rt, got)
		}
	}
}

func TestLogoutFromLMSSkipsIdP(t *testing.T) {
	setLogoutConfig(t)
	w := runLogout(t, "/auth/oidc/logout/lms", Handler.LogoutFromLMS)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://lms.test/" {
		t.Fatalf("status=%d Location=%q want 302 http://lms.test/", w.Code, w.Header().Get("Location"))
	}
}

func TestLogoutClearIsPlainPage(t *testing.T) {
	w := runLogout(t, "/auth/oidc/logout/clear", Handler.LogoutClear)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", w.Code)
	}
	if body := w.Body.String(); body != "<!doctype html><title>signed out</title>" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestLogoutRoutesRejectForeignSource(t *testing.T) {
	setLogoutConfig(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Handler{}.InitRoutes(router)
	for _, tc := range []struct {
		path, header, value string
		want                int
	}{
		{"/auth/oidc/logout/lk", "Origin", "https://evil.example", http.StatusForbidden},
		{"/auth/oidc/logout/lms", "Referer", "https://evil.example/x", http.StatusForbidden},
		{"/auth/oidc/logout/clear", "Origin", "null", http.StatusForbidden},
		{"/auth/oidc/logout/lk", "Referer", "http://lk.test/home", http.StatusFound},
		{"/auth/oidc/logout/lms", "Referer", "http://apps.lms.test/learning", http.StatusFound},
		{"/auth/oidc/logout/rs", "", "", http.StatusFound},
		{"/auth/oidc/logout/clear", "", "", http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.header != "" {
			req.Header.Set(tc.header, tc.value)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s %s=%q: status=%d want %d", tc.path, tc.header, tc.value, w.Code, tc.want)
		}
	}
}
