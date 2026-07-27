package server

import (
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func clearBFFSessionCookie(c *gin.Context) {
	secure := viper.GetBool("auth.refresh_cookie_secure")
	c.SetCookie(oidc.SessionCookieName, "", -1, "/", "", secure, true)
}

func sessionStillActive(sessions licensing.Gateway, sid string) bool {
	if sid == "" || sessions == nil {
		// No sid / no sessions store: cannot enforce; allow for backward compat.
		return true
	}
	sess, err := sessions.GetActiveSession(sid)
	return err == nil && sess != nil
}

func applyOidcSession(c *gin.Context, sessions licensing.Gateway) bool {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, err := oidc.ParseSessionToken(cookie); err == nil && claims.Sub != "" {
			if claims.Sid != "" && !sessionStillActive(sessions, claims.Sid) {
				clearBFFSessionCookie(c)
				return false
			}
			userID := claims.EdxUserID
			if userID == "" {
				userID = claims.Sub
			}
			c.Set("user_id", userID)
			c.Set("user_role", models.Role(claims.Role))
			c.Set("session_sid", claims.Sid)
			return true
		}
	}
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.Split(header, " ")
		if len(parts) == 2 {
			if claims, err := oidc.ParseSessionToken(parts[1]); err == nil && claims.Sub != "" {
				// Password JWT uses Id not Sub — only treat as BFF when typ/sub present.
				if claims.Sid != "" && !sessionStillActive(sessions, claims.Sid) {
					return false
				}
				userID := claims.EdxUserID
				if userID == "" {
					userID = claims.Sub
				}
				c.Set("user_id", userID)
				c.Set("user_role", models.Role(claims.Role))
				c.Set("session_sid", claims.Sid)
				return true
			}
		}
	}
	return false
}

func TokenAuthMiddleware(sessions licensing.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/internal/lms/") || strings.HasPrefix(path, "/auth/oidc/") {
			c.Next()
			return
		}
		if path == "/projectPage/public" ||
			(c.Request.Method == "GET" && strings.HasSuffix(path, "/preview") && strings.HasPrefix(path, "/projectPage/")) {
			c.Next()
			return
		}
		// Public RS3 activation / device-link / addon delivery (no BFF session).
		if path == "/v1/activate" ||
			path == "/v1/seats/deactivate" ||
			path == "/v1/device/link/start" ||
			path == "/v1/device/link/poll" ||
			path == "/addon/manifest.json" ||
			path == "/addon/paid-addon.js.enc" ||
			path == "/health/licensing" {
			c.Next()
			return
		}
		authMode := strings.ToLower(strings.TrimSpace(viper.GetString("auth.mode")))
		lmsDbMode := authMode == "lms_db"
		oidcBff := authMode == "oidc_bff" || viper.GetBool("oidc.enabled")
		lmsFallback := viper.GetBool("auth.lmsPasswordFallback") || lmsDbMode

		if lmsDbMode || (oidcBff && lmsFallback) {
			if applyOidcSession(c, sessions) {
				c.Next()
				return
			}
			// Fall through to JWT (LMS email/password login).
		} else if oidcBff {
			if applyOidcSession(c, sessions) {
				c.Next()
				return
			}
			c.Set("user_id", "0")
			c.Set("user_role", models.Anonymous)
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		cookie, gerTokenErr := c.Cookie("refresh_token")
		if gerTokenErr == nil {
			c.Set("refresh_token", cookie)
		} else {
			c.Set("refresh_token", "")
		}
		if header == "" {
			c.Set("user_id", "0")
			c.Set("user_role", models.Anonymous)
			c.Next()
			return
		}
		headerParts := strings.Split(header, " ")
		if len(headerParts) != 2 {
			graphql.AddError(c, &gqlerror.Error{
				Path:    graphql.GetPath(c),
				Message: "invalid authorization header format",
				Extensions: map[string]interface{}{
					"code": "401",
				},
			})
			c.Abort()
			return
		}
		data, err := jwt.ParseWithClaims(headerParts[1], &models.UserClaims{},
			func(token *jwt.Token) (interface{}, error) {
				return []byte(viper.GetString("auth.access_signing_key")), nil
			})

		if err != nil {
			c.AbortWithStatusJSON(401, err)
			return
		}

		claims, ok := data.Claims.(*models.UserClaims)
		if !ok {
			graphql.AddError(c, &gqlerror.Error{
				Path:    graphql.GetPath(c),
				Message: "token claims are not of type *StandardClaims",
				Extensions: map[string]interface{}{
					"code": "401",
				},
			})
			c.Abort()
			return
		}
		if claims.Sid != "" && !sessionStillActive(sessions, claims.Sid) {
			c.AbortWithStatusJSON(401, gin.H{"error": "SESSION_NOT_FOUND", "code": "SESSION_NOT_FOUND"})
			return
		}
		c.Set("user_id", claims.Id)
		c.Set("user_role", claims.Role)
		c.Set("session_sid", claims.Sid)
		c.Next()
	}
}
