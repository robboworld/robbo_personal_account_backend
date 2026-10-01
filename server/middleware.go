package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var errInvalidTokenAlg = errors.New("unexpected token alg")

func clearBFFSessionCookie(c *gin.Context) {
	oidc.ClearHTTPOnlyCookie(c, oidc.SessionCookieName)
}

// sessionOwnedBy reports whether sid is an active session of userID. Binding the token's
// user to the session row stops a forged or cross-used token from borrowing another sid.
func sessionOwnedBy(sessions licensing.Gateway, sid, userID string) bool {
	if sid == "" || userID == "" || sessions == nil {
		return false
	}
	sess, err := sessions.GetActiveSession(sid)
	return err == nil && sess != nil && sess.LmsUserID == userID
}

func abortIfUserInactive(c *gin.Context, userID string) bool {
	if userID == "" || userID == "0" {
		return false
	}
	active, err := lmsdb.IsUserActiveCached(userID)
	if err != nil {
		// Fail closed without logging the user out: LMS is unreachable, not the account disabled.
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "lms_unavailable",
			"code":  "LMS_UNAVAILABLE",
		})
		return true
	}
	if active {
		return false
	}
	clearBFFSessionCookie(c)
	body := gin.H{
		"error": "user account is disabled",
		"code":  "USER_INACTIVE",
	}
	if ban := moderation.LookupPublicBanInfo(userID); ban != nil {
		body["ban"] = moderation.PublicBanJSON(ban)
	}
	c.AbortWithStatusJSON(http.StatusForbidden, body)
	return true
}

func applyOidcSession(c *gin.Context, sessions licensing.Gateway) bool {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, err := oidc.ParseSessionToken(cookie); err == nil && claims.Sub != "" {
			userID := claims.EdxUserID
			if userID == "" {
				userID = claims.Sub
			}
			if !sessionOwnedBy(sessions, claims.Sid, userID) {
				clearBFFSessionCookie(c)
				return false
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
				userID := claims.EdxUserID
				if userID == "" {
					userID = claims.Sub
				}
				if !sessionOwnedBy(sessions, claims.Sid, userID) {
					return false
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

func proceedIfActive(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, _ := userID.(string)
	if abortIfUserInactive(c, id) {
		return
	}
	c.Next()
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
		if c.Request.Method == "GET" && (path == "/api/join/preview" ||
			(strings.HasPrefix(path, "/api/join/") && strings.HasSuffix(path, "/preview"))) {
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
				proceedIfActive(c)
				return
			}
			// Fall through to JWT (LMS email/password login).
		} else if oidcBff {
			if applyOidcSession(c, sessions) {
				proceedIfActive(c)
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
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, errInvalidTokenAlg
				}
				return []byte(viper.GetString("auth.access_signing_key")), nil
			})

		if err != nil {
			// Generic body: the parser error text is not for clients.
			c.AbortWithStatusJSON(401, gin.H{"error": "INVALID_TOKEN", "code": "INVALID_TOKEN"})
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
		if !sessionOwnedBy(sessions, claims.Sid, claims.Id) {
			c.AbortWithStatusJSON(401, gin.H{"error": "SESSION_NOT_FOUND", "code": "SESSION_NOT_FOUND"})
			return
		}
		c.Set("user_id", claims.Id)
		c.Set("user_role", claims.Role)
		c.Set("session_sid", claims.Sid)
		proceedIfActive(c)
	}
}
