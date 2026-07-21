package http

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/spf13/viper"
)

type Handler struct {
	cfg    *oidc.Config
	portal portalgateway.Gateway
}

func NewHandler(portal portalgateway.Gateway) (Handler, error) {
	cfg, err := oidc.LoadConfig()
	if err != nil {
		return Handler{}, err
	}
	return Handler{cfg: cfg, portal: portal}, nil
}

func (h Handler) InitRoutes(router *gin.Engine) {
	g := router.Group("/auth/oidc")
	{
		g.GET("/start", h.Start)
		g.GET("/callback", h.Callback)
		g.GET("/logout", h.Logout)
		g.GET("/status", h.Status)
	}
}

// Start initiates the OIDC Authorization Code + PKCE flow.
// By default uses prompt=none (silent SSO). When the IdP has no active session,
// the callback returns login_required; the frontend or next redirect should call
// /auth/oidc/start?prompt=login to show the IdP login page.
func (h Handler) Start(c *gin.Context) {
	returnTo := c.DefaultQuery("return_to", "/home")
	prompt := c.DefaultQuery("prompt", "none")
	if prompt != "none" && prompt != "login" && prompt != "consent" {
		prompt = "none"
	}
	entry, err := oidc.NewPKCEForReturn(returnTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pkce_init_failed"})
		return
	}
	authURL, err := url.Parse(h.cfg.AuthorizationEndpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid_authorization_endpoint"})
		return
	}
	q := authURL.Query()
	q.Set("client_id", h.cfg.ClientID)
	q.Set("redirect_uri", h.cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", h.cfg.Scopes)
	q.Set("state", entry.State)
	q.Set("nonce", entry.Nonce)
	q.Set("code_challenge", entry.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("prompt", prompt)
	authURL.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, authURL.String())
}

// Status returns whether the current request has an active BFF session.
func (h Handler) Status(c *gin.Context) {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, err := oidc.ParseSessionToken(cookie); err == nil && claims.Sub != "" {
			c.JSON(http.StatusOK, authStatusPayload(true, claims.Sub, claims.Email, claims.EdxUserID, claims.Role))
			return
		}
	}
	c.JSON(http.StatusOK, authStatusPayload(false, "", "", "", 0))
}

func authStatusPayload(authenticated bool, sub, email, edxUserID string, role uint) gin.H {
	return gin.H{
		"authenticated":         authenticated,
		"sub":                   sub,
		"email":                 email,
		"edx_user_id":           edxUserID,
		"role":                  role,
		"auth_mode":             viper.GetString("auth.mode"),
		"legacy_auth":           viper.GetBool("legacyPostgres.enabled"),
		"lms_password_fallback": viper.GetBool("auth.lmsPasswordFallback") ||
			strings.EqualFold(viper.GetString("auth.mode"), "lms_db"),
		"oidc_enabled": viper.GetBool("oidc.enabled"),
	}
}

// Logout clears the BFF session cookie and redirects to the IdP end_session endpoint.
func (h Handler) Logout(c *gin.Context) {
	returnTo := c.DefaultQuery("return_to", "")
	// clear BFF cookie
	c.SetCookie(oidc.SessionCookieName, "", -1, "/", "", false, true)

	logoutEndpoint := viper.GetString("oidc.logoutEndpoint")
	if logoutEndpoint == "" {
		// no IdP logout configured — just redirect to frontend
		frontend := viper.GetString("oidc.frontendBaseUrl")
		if frontend == "" {
			frontend = "http://localhost:3030"
		}
		target := frontend + "/login"
		if returnTo != "" {
			target = frontend + returnTo
		}
		c.Redirect(http.StatusFound, target)
		return
	}
	logoutURL, err := url.Parse(logoutEndpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid_logout_endpoint"})
		return
	}
	postLogout := viper.GetString("oidc.postLogoutRedirectUri")
	if returnTo != "" && strings.HasPrefix(returnTo, "http") {
		postLogout = returnTo
	}
	if postLogout != "" {
		q := logoutURL.Query()
		q.Set("post_logout_redirect_uri", postLogout)
		logoutURL.RawQuery = q.Encode()
	}
	c.Redirect(http.StatusFound, logoutURL.String())
}

func (h Handler) Callback(c *gin.Context) {
	if errParam := c.Query("error"); errParam != "" {
		// login_required means the IdP has no active session → retry with interactive login
		if errParam == "login_required" || errParam == "interaction_required" {
			state := c.Query("state")
			returnTo := "/home"
			if entry, ok := oidc.ConsumePKCE(state); ok {
				returnTo = entry.ReturnTo
			}
			startURL := fmt.Sprintf("/auth/oidc/start?prompt=login&return_to=%s",
				url.QueryEscape(returnTo))
			c.Redirect(http.StatusFound, startURL)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errParam,
			"error_description": c.Query("error_description"),
		})
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_code_or_state"})
		return
	}
	entry, ok := oidc.ConsumePKCE(state)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}
	tr, err := h.cfg.ExchangeCode(code, entry.CodeVerifier)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "token_exchange_failed", "details": err.Error()})
		return
	}
	claims, err := h.cfg.ValidateIDToken(tr.IDToken, entry.Nonce)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	edxUserID := ""
	role := models.Student
	if profile, err := lookupLMSProfileByEmail(claims.Email); err == nil && profile != nil {
		edxUserID = strconv.FormatInt(profile.ID, 10)
		role = lmsRoleFromProfile(profile)
		touchLastLogin(profile.ID)
	} else {
		role = roleCodeToModel(inferRoleCodeFromIDToken(tr.IDToken))
	}
	session, err := oidc.IssueSessionToken(claims.Sub, edxUserID, claims.Email, uint(role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	secure := viper.GetBool("auth.refresh_cookie_secure")
	c.SetCookie(oidc.SessionCookieName, session, oidc.SessionTTLSeconds(), "/", "", secure, true)
	target := entry.ReturnTo
	if target == "" {
		target = "/home"
	}
	if strings.HasPrefix(target, "http") {
		c.Redirect(http.StatusFound, target)
		return
	}
	frontend := viper.GetString("oidc.frontendBaseUrl")
	if frontend == "" {
		frontend = "http://localhost:3030"
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s%s", strings.TrimRight(frontend, "/"), target))
}

func inferRoleCodeFromIDToken(idToken string) string {
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return "student"
	}
	raw, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "student"
	}
	if claimBool(raw["is_superuser"]) {
		return "super_admin"
	}
	if claimBool(raw["is_staff"]) {
		return "teacher"
	}
	return "student"
}

func claimBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true") || b == "1"
	}
	return false
}

func roleCodeToModel(code string) models.Role {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "teacher":
		return models.Teacher
	case "parent":
		return models.Parent
	case "free_listener":
		return models.FreeListener
	case "unit_admin":
		return models.UnitAdmin
	case "super_admin":
		return models.SuperAdmin
	default:
		return models.Student
	}
}

func lookupLMSProfileByEmail(email string) (*lmsdb.AuthUserProfile, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return reader.LookupAuthUserProfileByEmail(email)
}

func lmsRoleFromProfile(u *lmsdb.AuthUserProfile) models.Role {
	if u == nil {
		return models.Student
	}
	if u.IsSuperuser {
		return models.SuperAdmin
	}
	if u.IsStaff {
		return models.Teacher
	}
	return models.Student
}

func touchLastLogin(userID int64) {
	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		log.Printf("oidc: writer for last_login: %v", err)
		return
	}
	defer writer.Close()
	if err := writer.TouchLastLogin(userID); err != nil {
		log.Printf("oidc: touch last_login for user %d: %v", userID, err)
	}
}
