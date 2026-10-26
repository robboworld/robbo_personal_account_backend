package oidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func deviceIDFor(cookie string) (string, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/sign-in", nil)
	if cookie != "" {
		c.Request.AddCookie(&http.Cookie{Name: DeviceCookieName, Value: cookie})
	}
	return DeviceID(c), w
}

func TestDeviceIDKeepsValidCookieAndRenewsIt(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	got, w := deviceIDFor(id)
	if got != id {
		t.Fatalf("got %q, want the existing id", got)
	}
	if set := w.Header().Get("Set-Cookie"); !strings.Contains(set, DeviceCookieName+"="+id) || !strings.Contains(set, "HttpOnly") {
		t.Fatalf("cookie not renewed: %q", set)
	}
}

func TestDeviceIDReplacesMissingOrForgedCookie(t *testing.T) {
	for _, cookie := range []string{"", "x", strings.Repeat("z", 32), "victim-session-key"} {
		got, _ := deviceIDFor(cookie)
		if !isDeviceID(got) || got == cookie {
			t.Fatalf("cookie %q: got %q, want a fresh 32-hex id", cookie, got)
		}
	}
}
