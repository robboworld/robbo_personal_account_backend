package server

import (
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func applyOidcSession(c *gin.Context) bool {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, err := oidc.ParseSessionToken(cookie); err == nil && claims.Sub != "" {
			userID := claims.EdxUserID
			if userID == "" {
				userID = claims.Sub
			}
			c.Set("user_id", userID)
			c.Set("user_role", models.Role(claims.Role))
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
				c.Set("user_id", userID)
				c.Set("user_role", models.Role(claims.Role))
				return true
			}
		}
	}
	return false
}

func TokenAuthMiddleware() gin.HandlerFunc {
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
		authMode := strings.ToLower(strings.TrimSpace(viper.GetString("auth.mode")))
		lmsDbMode := authMode == "lms_db"
		oidcBff := authMode == "oidc_bff" || viper.GetBool("oidc.enabled")
		lmsFallback := viper.GetBool("auth.lmsPasswordFallback") || lmsDbMode

		if lmsDbMode || (oidcBff && lmsFallback) {
			if applyOidcSession(c) {
				c.Next()
				return
			}
			// Fall through to JWT (LMS email/password login).
		} else if oidcBff {
			if applyOidcSession(c) {
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
			//c.JSON(http.StatusUnauthorized, gin.H{
			//	"error": "token not found",
			//})
			//c.Abort()
			c.Set("user_id", "0")
			c.Set("user_role", models.Anonymous)
			c.Next()
			return
		}
		headerParts := strings.Split(header, " ")
		if len(headerParts) != 2 {
			//c.JSON(http.StatusUnauthorized, gin.H{
			//	"error": "invalid authorization header format",
			//})
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
			//c.JSON(http.StatusUnauthorized, gin.H{
			//	"error": "token claims are not of type *StandardClaims",
			//})
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
		c.Set("user_id", claims.Id)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}
