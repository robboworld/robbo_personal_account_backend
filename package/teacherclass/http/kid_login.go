package http

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// Class sign-in for young students: architecture/ADR_CLASS_KID_LOGIN.md.
//
// The public routes are not on the general login limit (10 a minute per IP): a whole class
// sits behind one school address. They have their own limits per class page and per IP; the
// real guard is the per-child lock after wrong pictures, kept in the database.

const kidAuthMode = "class_kid"

// KidLoginHandler serves the teachers' kid accounts and the class sign-in page.
type KidLoginHandler struct {
	authDelegate auth.Delegate
	uc           teacherclass.UseCase
	sessions     licensing.Gateway
}

func NewKidLoginHandler(authDelegate auth.Delegate, uc teacherclass.UseCase, sessions licensing.Gateway) KidLoginHandler {
	return KidLoginHandler{authDelegate: authDelegate, uc: uc, sessions: sessions}
}

func (h KidLoginHandler) InitRoutes(router *gin.Engine) {
	api := router.Group("/api/teacher/classes/:classId")
	{
		api.GET("/kids", h.ListKids)
		api.POST("/kids", h.CreateKids)
		api.POST("/kids/cards", h.IssueCards)
		api.POST("/kids/:kidId/pictures", h.ResetPictures)
		api.POST("/kids/:kidId/unlock", h.Unlock)
		api.POST("/kids/:kidId/return", h.Return)
		api.POST("/kid-login/open", h.Open)
		api.POST("/kid-login/close", h.Close)
	}
	public := router.Group("/api/class-login")
	{
		public.POST("/card", h.LoginWithCard)
		public.GET("/code/:code", h.ByCode)
		public.GET("/:slug", h.Page)
		public.POST("/:slug", h.LoginWithPictures)
	}
}

func writeKidErr(c *gin.Context, err error) {
	switch {
	// 403, not 401: the SPA answers a 401 with a token refresh and a retry, and a retried
	// wrong secret would count twice towards the lock.
	case errors.Is(err, teacherclass.ErrBadKidSecret),
		errors.Is(err, teacherclass.ErrKidLoginClosed), errors.Is(err, teacherclass.ErrKidLocked),
		errors.Is(err, teacherclass.ErrKidConsentRequired):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, teacherclass.ErrLmsLinkedClass), errors.Is(err, teacherclass.ErrTooManyKids):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		writeErr(c, err)
	}
}

func (h KidLoginHandler) teacher(c *gin.Context) (string, models.Role, bool) {
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", 0, false
	}
	if err := h.authDelegate.UserAccess(role, teacherRoles(), c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", 0, false
	}
	return userID, role, true
}

func (h KidLoginHandler) ListKids(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	out, err := h.uc.ListKids(userID, role, c.Param("classId"))
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h KidLoginHandler) CreateKids(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	var body struct {
		Names   []string `json:"names"`
		Consent bool     `json:"consent"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	kids, err := h.uc.CreateKids(userID, role, c.Param("classId"), body.Names, body.Consent)
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"kids": kids})
}

func (h KidLoginHandler) IssueCards(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	var body struct {
		UserIDs []string `json:"userIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	cards, err := h.uc.IssueKidCards(userID, role, c.Param("classId"), body.UserIDs)
	if err != nil {
		writeKidErr(c, err)
		return
	}
	// The tokens are shown once: no cache may keep them.
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

func (h KidLoginHandler) ResetPictures(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	kid, err := h.uc.ResetKidPictures(userID, role, c.Param("classId"), c.Param("kidId"))
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, kid)
}

func (h KidLoginHandler) Unlock(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	if err := h.uc.UnlockKid(userID, role, c.Param("classId"), c.Param("kidId")); err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h KidLoginHandler) Return(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	if err := h.uc.ReturnKid(userID, role, c.Param("classId"), c.Param("kidId")); err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h KidLoginHandler) Open(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	var body struct {
		Minutes int `json:"minutes"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			writeErr(c, teacherclass.ErrBadRequest)
			return
		}
	}
	room, err := h.uc.OpenKidLogin(userID, role, c.Param("classId"), body.Minutes)
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (h KidLoginHandler) Close(c *gin.Context) {
	userID, role, ok := h.teacher(c)
	if !ok {
		return
	}
	room, err := h.uc.CloseKidLogin(userID, role, c.Param("classId"))
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

// Limits of the public routes, per process. With several backends each counts its own share;
// the per-child lock in the database holds whatever the number.
var (
	kidPageLimiter = newKeyedLimiter(120, time.Minute) // sign-ins per class page
	kidIPLimiter   = newKeyedLimiter(300, time.Minute) // sign-ins and page reads per IP
	kidCardLimiter = newKeyedLimiter(30, time.Minute)  // card sign-ins per IP
	// Room codes per IP: a classroom mistypes a few, guessing 10^6 codes takes weeks.
	kidCodeLimiter  = newKeyedLimiter(30, time.Minute)
	errKidRateLimit = gin.H{"error": "too_many_attempts", "code": "RATE_LIMITED"}
)

func (h KidLoginHandler) Page(c *gin.Context) {
	if !kidIPLimiter.allow(c.ClientIP()) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, errKidRateLimit)
		return
	}
	page, err := h.uc.KidLoginPage(c.Param("slug"))
	if err != nil {
		writeKidErr(c, err)
		return
	}
	// Children's names: no shared cache keeps them after the lesson.
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

// ByCode turns the six digits on the projector into the class page (the short /k address).
func (h KidLoginHandler) ByCode(c *gin.Context) {
	if !kidCodeLimiter.allow(c.ClientIP()) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, errKidRateLimit)
		return
	}
	slug, err := h.uc.KidRoomByCode(c.Param("code"))
	if err != nil {
		writeKidErr(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"classPage": "/c/" + slug})
}

func (h KidLoginHandler) LoginWithPictures(c *gin.Context) {
	slug := c.Param("slug")
	if !kidIPLimiter.allow(c.ClientIP()) || !kidPageLimiter.allow(slug) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, errKidRateLimit)
		return
	}
	var body struct {
		UserID   string `json:"userId"`
		Pictures []int  `json:"pictures"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	sess, err := h.uc.KidLoginWithPictures(slug, body.UserID, body.Pictures)
	if err != nil {
		writeKidErr(c, err)
		return
	}
	h.startSession(c, sess)
}

func (h KidLoginHandler) LoginWithCard(c *gin.Context) {
	if !kidCardLimiter.allow(c.ClientIP()) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, errKidRateLimit)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	sess, err := h.uc.KidLoginWithCard(body.Token)
	if err != nil {
		writeKidErr(c, err)
		return
	}
	h.startSession(c, sess)
}

// startSession issues the ЛК session like a password sign-in, for the length of a lesson.
// Whoever was signed in on this browser before (a shared school computer) is signed out.
func (h KidLoginHandler) startSession(c *gin.Context, sess *teacherclass.KidSession) {
	if !auth.IsOidcBffMode() || h.sessions == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "class_login_unavailable"})
		return
	}
	if cookie, err := c.Cookie(oidc.SessionCookieName); err == nil && cookie != "" {
		if claims, parseErr := oidc.ParseSessionToken(cookie); parseErr == nil && claims.Sid != "" {
			_ = h.sessions.RevokeSession(claims.Sid)
		}
	}
	login, err := licensing.AcquireLoginSession(
		h.sessions, sess.UserID, kidAuthMode,
		licensing.LoginClient{UserAgent: c.Request.UserAgent(), IPAddress: c.ClientIP(), DeviceID: oidc.DeviceID(c)},
		sess.TTL, models.Student,
	)
	if err != nil {
		log.Printf("class login: session for %s: %v", sess.UserID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_create_failed"})
		return
	}
	sub := sess.Username
	if sub == "" {
		sub = sess.Email
	}
	token, err := oidc.IssueSessionTokenTTL(sub, sess.UserID, sess.Email, uint(models.Student), login.SessionKey, sess.TTL)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	oidc.SetHTTPOnlyCookie(c, oidc.SessionCookieName, token, int(sess.TTL/time.Second))
	out := gin.H{
		"ok":        true,
		"userId":    sess.UserID,
		"classId":   sess.ClassID,
		"name":      sess.Name,
		"avatar":    sess.Avatar,
		"expiresAt": time.Now().Add(sess.TTL).UTC(),
	}
	if sess.LoginSlug != "" {
		out["classPage"] = "/c/" + sess.LoginSlug
	}
	c.JSON(http.StatusOK, out)
}

// keyedLimiter allows limit events per key in a sliding window.
type keyedLimiter struct {
	mu        sync.Mutex
	hits      map[string][]time.Time
	limit     int
	window    time.Duration
	lastSweep time.Time
}

func newKeyedLimiter(limit int, window time.Duration) *keyedLimiter {
	return &keyedLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *keyedLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > l.window {
		for k, ts := range l.hits {
			if len(ts) == 0 || !ts[len(ts)-1].After(cutoff) {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
