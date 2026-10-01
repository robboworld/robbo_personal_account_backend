package http

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/streak"
	"github.com/spf13/viper"
)

type Handler struct {
	cfg      *oidc.Config
	portal   portalgateway.Gateway
	sessions licensing.Gateway
	streak   streak.UseCase
}

func NewHandler(portal portalgateway.Gateway, sessions licensing.Gateway, streakUC streak.UseCase) (Handler, error) {
	cfg, err := oidc.LoadConfig()
	if err != nil {
		return Handler{}, err
	}
	return Handler{cfg: cfg, portal: portal, sessions: sessions, streak: streakUC}, nil
}

func (h Handler) InitRoutes(router *gin.Engine) {
	g := router.Group("/auth/oidc")
	{
		g.GET("/start", h.Start)
		g.GET("/callback", h.Callback)
		g.GET("/logout", h.Logout)
		g.GET("/logout/lk", h.LogoutFromLK)
		g.GET("/logout/rs", h.LogoutFromRS)
		g.GET("/logout/lms", h.LogoutFromLMS)
		g.GET("/logout/clear", h.LogoutClear)
		g.GET("/status", h.Status)
		g.POST("/verify-credentials", h.VerifyCredentials)
		g.POST("/password-login", h.PasswordLogin)
	}
}

func browserAuthorizationEndpoint(endpoint string, c *gin.Context) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	browserHost := parsed.Hostname()
	if browserHost == "host.docker.internal" {
		browserHost = "localhost"
	}
	if reqHost := c.Request.Host; reqHost != "" {
		if hostname, _, splitErr := net.SplitHostPort(reqHost); splitErr == nil && hostname != "" {
			if hostname != "localhost" && hostname != "127.0.0.1" {
				browserHost = hostname
			}
		}
	}

	port := parsed.Port()
	if port == "" {
		// Local mock IdP lives on :8081 when the authorize URL omits an explicit port.
		// Tutor / real IdPs use scheme defaults (:80 / :443) — do not force 8081.
		switch strings.ToLower(browserHost) {
		case "localhost", "127.0.0.1", "::1":
			port = "8081"
		default:
			parsed.Host = browserHost
			return parsed.String()
		}
	}
	parsed.Host = net.JoinHostPort(browserHost, port)
	return parsed.String()
}

// Start initiates the OIDC Authorization Code + PKCE flow.
// By default uses prompt=none (silent SSO). When the IdP has no active session,
// the callback returns login_required; the frontend or next redirect should call
// /auth/oidc/start?prompt=login to show the IdP login page.
func (h Handler) Start(c *gin.Context) {
	returnTo := oidc.SanitizeReturnTo(c.DefaultQuery("return_to", "/home"))
	prompt := c.DefaultQuery("prompt", "none")
	if prompt != "none" && prompt != "login" && prompt != "consent" {
		prompt = "none"
	}
	kickOtherSessions := c.Query("kick_other_sessions") == "1" || c.Query("kick_other_sessions") == "true"
	entry, err := oidc.NewPKCEForReturnWithPromptAndKick(returnTo, prompt, kickOtherSessions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pkce_init_failed"})
		return
	}
	authURL, err := url.Parse(browserAuthorizationEndpoint(h.cfg.AuthorizationEndpoint, c))
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
	if prompt == "login" {
		q.Set("max_age", "0")
	}
	authURL.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, authURL.String())
}

// Status returns whether the current request has an active BFF session.
func (h Handler) Status(c *gin.Context) {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, err := oidc.ParseSessionToken(cookie); err == nil && claims.Sub != "" {
			if claims.Sid == "" || h.sessions == nil {
				oidc.ClearHTTPOnlyCookie(c, oidc.SessionCookieName)
				c.JSON(http.StatusOK, authStatusPayload(false, "", "", "", 0))
				return
			}
			if sess, sErr := h.sessions.GetActiveSession(claims.Sid); sErr != nil || sess == nil {
				oidc.ClearHTTPOnlyCookie(c, oidc.SessionCookieName)
				c.JSON(http.StatusOK, authStatusPayload(false, "", "", "", 0))
				return
			}
			c.JSON(http.StatusOK, authStatusPayload(true, claims.Sub, claims.Email, claims.EdxUserID, claims.Role))
			return
		}
	}
	c.JSON(http.StatusOK, authStatusPayload(false, "", "", "", 0))
}

// VerifyCredentials checks LMS MySQL email/username + password before starting OIDC.
// Does not issue a session — used by the LK /login form to reject unknown users early.
func (h Handler) VerifyCredentials(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}
	u, errCode, status := verifyLMSCredentials(loginName(body.Email, body.Username), body.Password)
	if errCode != "" {
		c.JSON(status, gin.H{"ok": false, "error": errCode})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"email": u.Email,
	})
}

// PasswordLogin verifies LMS credentials and issues the BFF session cookie without
// redirecting through the IdP login UI (avoids a mock/LMS page flash on /login).
func (h Handler) PasswordLogin(c *gin.Context) {
	var body struct {
		Email             string `json:"email"`
		Username          string `json:"username"`
		Password          string `json:"password"`
		ReturnTo          string `json:"return_to"`
		KickOtherSessions bool   `json:"kickOtherSessions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}
	u, errCode, status := verifyLMSCredentials(loginName(body.Email, body.Username), body.Password)
	if errCode != "" {
		c.JSON(status, gin.H{"ok": false, "error": errCode})
		return
	}

	edxUserID := strconv.FormatInt(u.ID, 10)
	role := models.Student
	if u.IsSuperuser {
		role = models.SuperAdmin
	} else if u.IsStaff {
		role = models.Teacher
	}
	touchLastLogin(u.ID)

	sid := ""
	if h.sessions != nil {
		ttl := time.Duration(oidc.SessionTTLSeconds()) * time.Second
		ip := c.ClientIP()
		if body.KickOtherSessions {
			if kickErr := licensing.KickOtherSessions(h.sessions, edxUserID); kickErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "session_create_failed"})
				return
			}
		}
		sess, createErr := licensing.AcquireLoginSession(
			h.sessions, edxUserID, "oidc_bff", c.Request.UserAgent(), ip, ttl, role,
		)
		if createErr != nil {
			if errors.Is(createErr, licensing.ErrSessionLimitReached) {
				c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "session_limit_reached"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "session_create_failed"})
			return
		}
		sid = sess.SessionKey
	}

	sub := u.Username
	if sub == "" {
		sub = u.Email
	}
	session, err := oidc.IssueSessionToken(sub, edxUserID, u.Email, uint(role), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "session_issue_failed"})
		return
	}
	oidc.SetHTTPOnlyCookie(c, oidc.SessionCookieName, session, oidc.SessionTTLSeconds())

	returnTo := oidc.SanitizeReturnTo(body.ReturnTo)
	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"email":     u.Email,
		"return_to": returnTo,
	})
}

func loginName(email, username string) string {
	if login := strings.TrimSpace(email); login != "" {
		return login
	}
	return strings.TrimSpace(username)
}

// dummyPasswordHash keeps unknown-user logins as slow as real ones (no timing oracle).
const dummyPasswordHash = "pbkdf2_sha256$870000$robbodummysalt$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// verifyLMSCredentials checks login/password against LMS auth_user. Unknown users and wrong
// passwords both return invalid_credentials, and an inactive account is reported only after
// the password matched, so the endpoints do not reveal which accounts exist.
func verifyLMSCredentials(login, password string) (*lmsdb.AuthUserLogin, string, int) {
	if login == "" || password == "" {
		return nil, "missing_credentials", http.StatusBadRequest
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, "lms_unavailable", http.StatusServiceUnavailable
	}
	defer reader.Close()

	u, err := reader.LookupAuthUserForLogin(login)
	if err != nil {
		return nil, "lms_unavailable", http.StatusServiceUnavailable
	}
	if u == nil {
		lmsdb.VerifyDjangoPassword(password, dummyPasswordHash)
		return nil, "invalid_credentials", http.StatusUnauthorized
	}
	if !lmsdb.VerifyDjangoPassword(password, u.Password) {
		return nil, "invalid_credentials", http.StatusUnauthorized
	}
	if !u.IsActive {
		return nil, "user_inactive", http.StatusForbidden
	}
	return u, "", http.StatusOK
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

// resolveLogoutTarget builds a post-logout redirect when IdP end_session is not used.
func resolveLogoutTarget(frontend, returnTo string) string {
	frontend = strings.TrimRight(frontend, "/")
	if strings.TrimSpace(returnTo) == "" {
		return frontend + "/"
	}
	returnTo = oidc.SanitizeReturnTo(returnTo)
	if strings.HasPrefix(returnTo, "http://") || strings.HasPrefix(returnTo, "https://") {
		return returnTo
	}
	if strings.HasPrefix(returnTo, "/") {
		return frontend + returnTo
	}
	return frontend + "/"
}

// LogoutFromLK: clear BFF, then LMS logout, land on LK.
func (h Handler) LogoutFromLK(c *gin.Context) {
	h.finishLogout(c, false, oidc.ProductLanding("lk"))
}

// LogoutFromRS: clear BFF and IdP session, return to Scratch editor.
// Top-level redirect only (no popup bridge): RS must not open extra browser windows.
func (h Handler) LogoutFromRS(c *gin.Context) {
	h.finishLogoutTopLevel(c, rsLogoutReturnTo(c.Query("return_to")))
}

// rsLogoutReturnTo keeps an allowlisted absolute return_to (e.g. a Scratch editor URL);
// anything else, including relative LK paths, falls back to the RS landing.
func rsLogoutReturnTo(raw string) string {
	if raw = strings.TrimSpace(raw); raw != "" {
		if safe := oidc.SanitizeReturnTo(raw); strings.HasPrefix(safe, "http://") || strings.HasPrefix(safe, "https://") {
			return safe
		}
	}
	return oidc.ProductLanding("rs")
}

// LogoutFromLMS: LMS already logged out; clear BFF and return to LMS landing.
func (h Handler) LogoutFromLMS(c *gin.Context) {
	h.finishLogout(c, true, oidc.ProductLanding("lms"))
}

// LogoutClear drops the BFF cookie and returns 200 HTML (Open edX IDA logout iframe).
func (h Handler) LogoutClear(c *gin.Context) {
	h.revokeBFFSession(c)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, "<!doctype html><title>signed out</title>")
}

// Logout keeps query-param API for callers that still pass return_to / skip_idp.
func (h Handler) Logout(c *gin.Context) {
	fixAmpInQuery(c)
	returnTo := strings.TrimSpace(c.DefaultQuery("return_to", ""))
	if returnTo != "" {
		returnTo = oidc.SanitizeReturnTo(returnTo)
	}
	skipIdP := strings.EqualFold(strings.TrimSpace(c.Query("skip_idp")), "1") ||
		strings.EqualFold(strings.TrimSpace(c.Query("local_only")), "1")
	h.finishLogout(c, skipIdP, returnTo)
}

func fixAmpInQuery(c *gin.Context) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return
	}
	if strings.Contains(c.Request.URL.RawQuery, "&amp;") {
		c.Request.URL.RawQuery = strings.ReplaceAll(c.Request.URL.RawQuery, "&amp;", "&")
	}
}

func (h Handler) revokeBFFSession(c *gin.Context) {
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" && h.sessions != nil {
		if claims, parseErr := oidc.ParseSessionToken(cookie); parseErr == nil && claims.Sid != "" {
			_ = h.sessions.RevokeSession(claims.Sid)
		}
	}
	oidc.ClearHTTPOnlyCookie(c, oidc.SessionCookieName)
	oidc.ClearHTTPOnlyCookie(c, "refresh_token")
}

// idpLogoutURLWithRedirect builds the IdP /logout URL that returns the browser to redirectTarget.
// Open edX /logout reads redirect_url; nested query strings in that value get HTML-escaped
// (& → &amp;), so product landings must not contain '&'.
func idpLogoutURLWithRedirect(logoutEndpoint, redirectTarget string) (string, error) {
	logoutURL, err := url.Parse(logoutEndpoint)
	if err != nil {
		return "", err
	}
	q := logoutURL.Query()
	q.Set("redirect_url", redirectTarget)
	q.Set("post_logout_redirect_uri", redirectTarget)
	logoutURL.RawQuery = q.Encode()
	return logoutURL.String(), nil
}

// finishLogoutTopLevel clears BFF then redirects the same browser tab through IdP /logout.
func (h Handler) finishLogoutTopLevel(c *gin.Context, returnTo string) {
	h.finishLogout(c, false, returnTo)
}

// finishLogout clears the BFF session and redirects the same tab: through IdP /logout
// (which then returns to the product landing) unless skipIdP or no logout endpoint is set.
// No popup bridge: popups are blocked or detached (noopener) and LMS forbids framing /logout.
func (h Handler) finishLogout(c *gin.Context, skipIdP bool, returnTo string) {
	h.revokeBFFSession(c)

	logoutEndpoint := viper.GetString("oidc.logoutEndpoint")
	frontend := viper.GetString("oidc.frontendBaseUrl")
	if frontend == "" {
		frontend = "http://localhost:3030"
	}

	postLogout := resolveLogoutTarget(frontend, returnTo)
	if skipIdP || logoutEndpoint == "" {
		c.Redirect(http.StatusFound, postLogout)
		return
	}
	idpLogoutURL, err := idpLogoutURLWithRedirect(logoutEndpoint, postLogout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid_logout_endpoint"})
		return
	}
	c.Redirect(http.StatusFound, idpLogoutURL)
}

func (h Handler) Callback(c *gin.Context) {
	if errParam := c.Query("error"); errParam != "" {
		// login_required / interaction_required: IdP has no session.
		// Silent (prompt=none) → back to LK /login so AuthLayout stays visible.
		// Interactive (prompt=login) → retry start with prompt=login (IdP form).
		if errParam == "login_required" || errParam == "interaction_required" {
			state := c.Query("state")
			returnTo := "/home"
			prompt := "login"
			if entry, ok := oidc.ConsumePKCE(state); ok {
				returnTo = oidc.SanitizeReturnTo(entry.ReturnTo)
				if entry.Prompt != "" {
					prompt = entry.Prompt
				}
			}
			if prompt == "none" {
				// Stay on the originating product (RS / LMS / LK), do not dump onto LK /login.
				if strings.HasPrefix(returnTo, "http://") || strings.HasPrefix(returnTo, "https://") {
					c.Redirect(http.StatusFound, returnTo)
					return
				}
				frontend := viper.GetString("oidc.frontendBaseUrl")
				if frontend == "" {
					frontend = "http://localhost:3030"
				}
				q := url.Values{}
				q.Set("return_to", returnTo)
				q.Set("sso_attempted", "1")
				redirectURL := fmt.Sprintf("%s/login?%s", strings.TrimRight(frontend, "/"), q.Encode())
				c.Redirect(http.StatusFound, redirectURL)
				return
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
	// Consume PKCE before token exchange so concurrent/duplicate callbacks cannot
	// reuse the same verifier, and a burned auth code cannot be retried via refresh.
	// (IdP authorization codes are single-use: exchange then failed validate left users
	// on invalid_grant when refreshing the callback URL.)
	entry, ok := oidc.ConsumePKCE(state)
	if !ok {
		redirectOIDCLoginError(c, "/home", "auth_retry")
		return
	}
	tr, err := h.cfg.ExchangeCode(code, entry.CodeVerifier)
	if err != nil {
		log.Printf("oidc: token exchange failed: %v", err)
		redirectOIDCLoginError(c, entry.ReturnTo, "auth_retry")
		return
	}
	claims, err := h.cfg.ValidateIDToken(tr.IDToken, entry.Nonce)
	if err != nil {
		log.Printf("oidc: token validate failed: %v", err)
		redirectOIDCLoginError(c, entry.ReturnTo, "token_invalid")
		return
	}
	edxUserID := ""
	role := models.Student
	profile, lookupErr := lookupLMSProfileByEmail(claims.Email)
	if lookupErr != nil {
		log.Printf("oidc: LMS profile lookup failed: %v", lookupErr)
		redirectOIDCLoginError(c, entry.ReturnTo, "auth_retry")
		return
	}
	if profile != nil {
		if !profile.IsActive {
			frontend := viper.GetString("oidc.frontendBaseUrl")
			if frontend == "" {
				frontend = "http://localhost:3030"
			}
			q := url.Values{}
			q.Set("err", "user_inactive")
			edxID := strconv.FormatInt(profile.ID, 10)
			if ban := moderation.LookupPublicBanInfo(edxID); ban != nil {
				if ban.Reason != "" {
					q.Set("reason", ban.Reason)
				}
				if ban.IsPermanent {
					q.Set("permanent", "1")
				} else if ban.ExpiresAt != nil {
					q.Set("expiresAt", ban.ExpiresAt.UTC().Format(time.RFC3339))
				}
			}
			redirectURL := fmt.Sprintf("%s/login?%s", strings.TrimRight(frontend, "/"), q.Encode())
			c.Redirect(http.StatusFound, redirectURL)
			return
		}
		edxUserID = strconv.FormatInt(profile.ID, 10)
		role = lmsRoleFromProfile(profile)
		touchLastLogin(profile.ID)
		if h.streak != nil {
			if _, streakErr := h.streak.RecordVisit(edxUserID, "UTC"); streakErr != nil {
				log.Printf("oidc: record streak for user %s: %v", edxUserID, streakErr)
			}
		}
	} else {
		// Unknown email / no LMS user — do not create a LK session.
		frontend := viper.GetString("oidc.frontendBaseUrl")
		if frontend == "" {
			frontend = "http://localhost:3030"
		}
		q := url.Values{}
		q.Set("err", "user_not_found")
		redirectURL := fmt.Sprintf("%s/login?%s", strings.TrimRight(frontend, "/"), q.Encode())
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	sid := ""
	if edxUserID != "" && h.sessions != nil {
		ttl := time.Duration(oidc.SessionTTLSeconds()) * time.Second
		ip := c.ClientIP()
		if entry.KickOtherSessions {
			if kickErr := licensing.KickOtherSessions(h.sessions, edxUserID); kickErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "session_create_failed"})
				return
			}
		}
		sess, createErr := licensing.AcquireLoginSession(
			h.sessions, edxUserID, "oidc_bff", c.Request.UserAgent(), ip, ttl, role,
		)
		if createErr != nil {
			if errors.Is(createErr, licensing.ErrSessionLimitReached) {
				frontend := viper.GetString("oidc.frontendBaseUrl")
				if frontend == "" {
					frontend = "http://localhost:3030"
				}
				target := entry.ReturnTo
				if target == "" {
					target = "/home"
				}
				sep := "?"
				if strings.Contains(target, "?") {
					sep = "&"
				}
				redirectURL := fmt.Sprintf("%s%s%serr=session_limit_reached",
					strings.TrimRight(frontend, "/"), target, sep)
				if strings.HasPrefix(target, "http") {
					redirectURL = fmt.Sprintf("%s%serr=session_limit_reached", target, sep)
				}
				c.Redirect(http.StatusFound, redirectURL)
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session_create_failed"})
			return
		}
		sid = sess.SessionKey
	}

	session, err := oidc.IssueSessionToken(claims.Sub, edxUserID, claims.Email, uint(role), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	oidc.SetHTTPOnlyCookie(c, oidc.SessionCookieName, session, oidc.SessionTTLSeconds())
	target := oidc.SanitizeReturnTo(entry.ReturnTo)
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		c.Redirect(http.StatusFound, target)
		return
	}
	frontend := viper.GetString("oidc.frontendBaseUrl")
	if frontend == "" {
		frontend = "http://localhost:3030"
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s%s", strings.TrimRight(frontend, "/"), target))
}

// redirectOIDCLoginError sends the browser back to LK /login to start a fresh SSO attempt.
// Prefer redirect over JSON: browsers land on callback URLs and users refresh them.
func redirectOIDCLoginError(c *gin.Context, returnTo, errCode string) {
	frontend := viper.GetString("oidc.frontendBaseUrl")
	if frontend == "" {
		frontend = "http://localhost:3030"
	}
	q := url.Values{}
	q.Set("err", errCode)
	q.Set("sso_attempted", "1")
	if returnTo != "" {
		q.Set("return_to", oidc.SanitizeReturnTo(returnTo))
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s/login?%s", strings.TrimRight(frontend, "/"), q.Encode()))
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
