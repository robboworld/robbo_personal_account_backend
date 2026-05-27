package http

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	auth := router.Group("/auth")
	{
		auth.POST("/sign-up", h.SignUp)
		auth.POST("/sign-in", h.SignIn)
		auth.GET("/refresh", h.Refresh)
		auth.POST("/sign-out", h.SignOut)
		auth.GET("/check-auth", h.CheckAuth)
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

func (h *Handler) SignIn(c *gin.Context) {
	fmt.Println("SignIn")

	signInInput := &signInput{}
	if err := c.BindJSON(signInInput); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.delegate.SignIn(signInInput.Email, signInInput.Password, signInInput.Role)
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
	Company string `json:"company"`
}

func (h *Handler) SignUp(c *gin.Context) {
	fmt.Println("SignUp")

	body := &signUpBody{}

	if err := c.BindJSON(body); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	body.UserHTTP.Company = strings.TrimSpace(body.Company)

	accessToken, refreshToken, err := h.delegate.SignUp(&body.UserHTTP)
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
