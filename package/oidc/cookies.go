package oidc

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// CookieSameSite is Lax by default; AUTH_REFRESH_COOKIE_SAMESITE=none|strict|lax.
func CookieSameSite() http.SameSite {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_REFRESH_COOKIE_SAMESITE")))
	if v == "" {
		v = strings.ToLower(strings.TrimSpace(viper.GetString("auth.refresh_cookie_samesite")))
	}
	switch v {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

func cookieSecure() bool {
	if CookieSameSite() == http.SameSiteNoneMode {
		return true
	}
	return viper.GetBool("auth.refresh_cookie_secure")
}

// ApplyCookiePolicy sets SameSite for subsequent gin SetCookie calls.
func ApplyCookiePolicy(c *gin.Context) {
	if c == nil {
		return
	}
	c.SetSameSite(CookieSameSite())
}

// SetHTTPOnlyCookie writes a host-only HttpOnly cookie with the shared policy.
func SetHTTPOnlyCookie(c *gin.Context, name, value string, maxAge int) {
	if c == nil {
		return
	}
	ApplyCookiePolicy(c)
	c.SetCookie(name, value, maxAge, "/", "", cookieSecure(), true)
}

// ClearHTTPOnlyCookie expires a cookie using the same flags.
func ClearHTTPOnlyCookie(c *gin.Context, name string) {
	SetHTTPOnlyCookie(c, name, "", -1)
}
