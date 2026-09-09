package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
)

func csrfExemptPath(path string) bool {
	switch path {
	case "/auth/sign-in", "/auth/sign-up",
		"/auth/oidc/verify-credentials", "/auth/oidc/password-login",
		"/auth/refresh":
		return true
	}
	if strings.HasPrefix(path, "/internal/lms/") {
		return true
	}
	if path == "/v1/activate" ||
		path == "/v1/seats/deactivate" ||
		path == "/v1/device/link/start" ||
		path == "/v1/device/link/poll" {
		return true
	}
	return false
}

// CookieCSRFMiddleware requires X-Requested-With on cookie-authenticated mutations.
func CookieCSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if csrfExemptPath(path) {
			c.Next()
			return
		}
		authz := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") && strings.TrimSpace(authz[7:]) != "" {
			c.Next()
			return
		}
		_, sessErr := c.Cookie(oidc.SessionCookieName)
		_, refreshErr := c.Cookie("refresh_token")
		if sessErr != nil && refreshErr != nil {
			c.Next()
			return
		}
		if strings.TrimSpace(c.GetHeader("X-Requested-With")) == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf_header_required",
				"code":  "CSRF_HEADER_REQUIRED",
			})
			return
		}
		c.Next()
	}
}
