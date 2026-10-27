package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	notificationusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/notifications/usecase"
)

type Handler struct {
	authDelegate auth.Delegate
	useCase      notifications.UseCase
}

func NewNotificationHandler(authDelegate auth.Delegate, useCase notifications.UseCase) Handler {
	return Handler{authDelegate: authDelegate, useCase: useCase}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	group := router.Group("/api/notifications")
	group.GET("/feed", h.Feed)
	group.GET("/unread-count", h.UnreadCount)
	group.POST("/mark-personal-read", h.MarkPersonalRead)
	group.POST("/mark-announcement-read", h.MarkAnnouncementRead)
	group.POST("/mark-lms-read", h.MarkLMSRead)
	group.POST("/mark-all-read", h.MarkAllRead)
	group.POST("/admin/personal", h.AdminPersonal)
	group.POST("/admin/announcement", h.AdminAnnouncement)
	group.POST("/teacher/send", h.TeacherSend)
	group.GET("/teacher/quota", h.TeacherQuota)
}

func allRoles() []models.Role {
	return []models.Role{
		models.Student,
		models.Teacher,
		models.Parent,
		models.FreeListener,
		models.UnitAdmin,
		models.SuperAdmin,
	}
}

func (h *Handler) identity(c *gin.Context, roles []models.Role) (string, bool) {
	userID, _, ok := h.identityWithRole(c, roles)
	return userID, ok
}

func (h *Handler) identityWithRole(c *gin.Context, roles []models.Role) (string, models.Role, bool) {
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", role, false
	}
	if err := h.authDelegate.UserAccess(role, roles, c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", role, false
	}
	return userID, role, true
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, notificationusecase.ErrInvalidInput) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, notificationusecase.ErrUserNotFound) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, notificationusecase.ErrNotYourStudent) {
		status = http.StatusForbidden
	}
	if errors.Is(err, notifications.ErrTeacherDailyLimit) {
		status = http.StatusTooManyRequests
	}
	c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
}

func (h *Handler) Feed(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	feed, err := h.useCase.Feed(userID, limit, strings.TrimSpace(c.Query("source")))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, feed)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	counts, err := h.useCase.UnreadCount(userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

type idRequest struct {
	ID string `json:"id"`
}

func (h *Handler) MarkPersonalRead(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	var request idRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	if err := h.useCase.MarkPersonalRead(userID, request.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) MarkAnnouncementRead(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	var request idRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	if err := h.useCase.MarkAnnouncementRead(userID, request.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MarkLMSRead: an Open edX notification (feedKind "lms") read in the LK bell.
func (h *Handler) MarkLMSRead(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	var request idRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	if err := h.useCase.MarkLMSRead(userID, request.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	// Optional body {"source": "lk"|"lms"}: one tab of the bell; none = everything.
	var request struct {
		Source string `json:"source"`
	}
	_ = c.ShouldBindJSON(&request)
	if err := h.useCase.MarkAllRead(userID, strings.TrimSpace(request.Source)); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type personalRequest struct {
	RecipientUserID   string  `json:"recipientUserId"`
	RecipientUsername string  `json:"recipientUsername"`
	Title             string  `json:"title"`
	Body              string  `json:"body"`
	Kind              string  `json:"kind"`
	Severity          string  `json:"severity"`
	ActionURL         *string `json:"actionUrl"`
}

func (h *Handler) AdminPersonal(c *gin.Context) {
	_, ok := h.identity(c, []models.Role{models.UnitAdmin, models.SuperAdmin})
	if !ok {
		return
	}
	var request personalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	if err := h.useCase.SendPersonal(notifications.PersonalInput{
		RecipientUserID:   request.RecipientUserID,
		RecipientUsername: request.RecipientUsername,
		Title:             request.Title,
		Body:              request.Body,
		Kind:              request.Kind,
		Severity:          request.Severity,
		ActionURL:         request.ActionURL,
	}); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type announcementRequest struct {
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	AudienceJSON string  `json:"audienceJson"`
	ExpiresAt    *string `json:"expiresAt"`
}

func (h *Handler) AdminAnnouncement(c *gin.Context) {
	userID, ok := h.identity(c, []models.Role{models.SuperAdmin})
	if !ok {
		return
	}
	var request announcementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	var expiresAt *time.Time
	if request.ExpiresAt != nil && strings.TrimSpace(*request.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.ExpiresAt))
		if err != nil {
			writeError(c, notificationusecase.ErrInvalidInput)
			return
		}
		expiresAt = &parsed
	}
	if err := h.useCase.CreateAnnouncement(notifications.AnnouncementInput{
		Title:           request.Title,
		Body:            request.Body,
		AudienceJSON:    request.AudienceJSON,
		CreatedByUserID: userID,
		ExpiresAt:       expiresAt,
	}); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type teacherSendRequest struct {
	RecipientUserIDs []string `json:"recipientUserIds"`
	Title            string   `json:"title"`
	Body             string   `json:"body"`
}

// TeacherSend: a teacher messages students of their own classes (SuperAdmin: anyone).
// No kind, severity or link from the client: a teacher must not be able to put an arbitrary URL in front of children.
func (h *Handler) TeacherSend(c *gin.Context) {
	userID, role, ok := h.identityWithRole(c, []models.Role{models.Teacher, models.SuperAdmin})
	if !ok {
		return
	}
	var request teacherSendRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, notificationusecase.ErrInvalidInput)
		return
	}
	sent, quota, err := h.useCase.SendFromTeacher(notifications.TeacherMessageInput{
		SenderUserID:     userID,
		SenderRole:       role,
		RecipientUserIDs: request.RecipientUserIDs,
		Title:            request.Title,
		Body:             request.Body,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": sent, "quota": quota})
}

func (h *Handler) TeacherQuota(c *gin.Context) {
	userID, role, ok := h.identityWithRole(c, []models.Role{models.Teacher, models.SuperAdmin})
	if !ok {
		return
	}
	quota, err := h.useCase.TeacherQuota(userID, role)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, quota)
}
