package http

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
)

type Handler struct {
	delegate auth.Delegate
}

func NewAuthHandler(
	authDelegate auth.Delegate,
) Handler {
	return Handler{
		delegate: authDelegate,
	}
}

func (h *Handler) InitAuthRoutes(router *gin.Engine) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/sign-up", h.SignUp)
		authGroup.POST("/sign-in", h.SignIn)
		authGroup.GET("/refresh", h.Refresh)
		authGroup.POST("/sign-out", h.SignOut)
		authGroup.GET("/check-auth", h.CheckAuth)
		authGroup.GET("/sessions", h.ListSessions)
		authGroup.DELETE("/sessions/:id", h.RevokeSession)
	}
}

type signInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     uint   `json:"role"`
}

type signInResponse struct {
	AccessToken string `json:"accessToken"`
}

func clientInfoFromRequest(c *gin.Context) auth.ClientInfo {
	return auth.ClientInfo{
		UserAgent: c.Request.UserAgent(),
		IPAddress: clientIP(c),
	}
}

func clientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return c.Request.RemoteAddr
}

func (h *Handler) SignIn(c *gin.Context) {
	fmt.Println("SignIn")

	signInInput := &signInput{}
	if err := c.BindJSON(signInInput); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.delegate.SignIn(
		signInInput.Email, signInInput.Password, signInInput.Role, clientInfoFromRequest(c),
	)
	if err != nil {
		fmt.Println(err)
		ErrorHandling(err, c)
		return
	}

	setRefreshToken(refreshToken, c)

	c.JSON(http.StatusOK, signInResponse{
		AccessToken: accessToken,
	})
}

type signUpBody struct {
	models.UserHTTP
	PhoneNumber          string `json:"phone_number"`
	HonorCode            bool   `json:"honor_code"`
	MarketingEmailsOptIn bool   `json:"marketing_emails_opt_in"`
}

func (h *Handler) SignUp(c *gin.Context) {
	fmt.Println("SignUp")

	body := &signUpBody{}

	if err := c.BindJSON(body); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	userCore := body.UserHTTP.ToCore()
	userCore.PhoneNumber = strings.TrimSpace(body.PhoneNumber)
	userCore.HonorCode = body.HonorCode
	userCore.MarketingOptIn = body.MarketingEmailsOptIn

	accessToken, refreshToken, err := h.delegate.SignUpCore(&userCore, clientInfoFromRequest(c))
	if err != nil {
		ErrorHandling(err, c)
		return
	}

	setRefreshToken(refreshToken, c)

	c.JSON(http.StatusOK, signInResponse{
		AccessToken: accessToken,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	fmt.Println("Refresh")

	refreshToken, err := getRefreshToken(c)
	if err != nil {
		ErrorHandling(err, c)
		return
	}

	newAccessToken, err := h.delegate.RefreshToken(refreshToken)
	if err != nil {
		fmt.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, signInResponse{
		AccessToken: newAccessToken,
	})
}

func (h *Handler) SignOut(c *gin.Context) {
	fmt.Println("SignOut")
	if refreshToken, err := getRefreshToken(c); err == nil {
		_ = h.delegate.SignOut(refreshToken)
	}
	setRefreshToken("", c)
	c.Status(http.StatusOK)
}

type userIdentity struct {
	Id   string `json:"id"`
	Role uint   `json:"role"`
}

func (h *Handler) CheckAuth(c *gin.Context) {
	fmt.Println("CheckAuth")
	userId, role, err := h.delegate.UserIdentity(c)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, &userIdentity{
		userId,
		uint(role),
	})
}

func (h *Handler) ListSessions(c *gin.Context) {
	userID, _, err := h.delegate.UserIdentity(c)
	if err != nil || userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sessions, err := h.delegate.ListSessions(userID)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	out := make([]gin.H, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionToJSON(s))
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

func (h *Handler) RevokeSession(c *gin.Context) {
	userID, _, err := h.delegate.UserIdentity(c)
	if err != nil || userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	if err := h.delegate.RevokeSessionByID(userID, sessionID); err != nil {
		ErrorHandling(err, c)
		return
	}
	c.Status(http.StatusNoContent)
}

func sessionToJSON(s *models.UserSessionCore) gin.H {
	out := gin.H{
		"id":         s.ID,
		"authMode":   s.AuthMode,
		"userAgent":  s.UserAgent,
		"ipAddress":  s.IPAddress,
		"createdAt":  s.CreatedAt.UTC().Format(time.RFC3339),
		"lastSeenAt": s.LastSeenAt.UTC().Format(time.RFC3339),
		"expiresAt":  s.ExpiresAt.UTC().Format(time.RFC3339),
	}
	return out
}

func ErrorHandling(err error, c *gin.Context) {
	switch {
	case errors.Is(err, auth.ErrEmailAlreadyExist):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "field": "email"})
	case errors.Is(err, auth.ErrUsernameAlreadyExist):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "field": "nickname"})
	case errors.Is(err, auth.ErrUserAlreadyExist):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrCompanyRequired):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrInvalidAccessToken), errors.Is(err, auth.ErrInvalidTypeClaims), errors.Is(err, auth.ErrTokenNotFound):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrSessionNotFound):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "code": "SESSION_NOT_FOUND"})
	case errors.Is(err, auth.ErrSessionLimitReached):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error": err.Error(),
			"code":  "SESSION_LIMIT_REACHED",
		})
	case errors.Is(err, auth.ErrLegacyAuthDisabled):
		c.AbortWithStatusJSON(http.StatusGone, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrUserNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrInvalidCredentials):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, auth.ErrUserInactive):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, http.ErrNoCookie):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func getRefreshToken(c *gin.Context) (refreshToken string, err error) {
	cookie, gerTokenErr := c.Cookie("refresh_token")
	if gerTokenErr != nil {
		if gerTokenErr == http.ErrNoCookie {
			log.Println("Error finding cookie: ", gerTokenErr)
			err = http.ErrNoCookie
		}
		err = gerTokenErr
		fmt.Println(err)
		return "", err
	}
	refreshToken = cookie
	return
}

func setRefreshToken(value string, c *gin.Context) {
	c.SetCookie(
		"refresh_token",
		value,
		60*60*24*7,
		"/",
		"",
		viper.GetBool("auth.refresh_cookie_secure"),
		true,
	)
}
