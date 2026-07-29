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
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	modusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/moderation/usecase"
)

// Handler serves SuperAdmin user ban/unban endpoints.
type Handler struct {
	authDelegate auth.Delegate
	delegate     moderation.Delegate
}

// NewHandler creates a moderation HTTP handler.
func NewHandler(authDelegate auth.Delegate, delegate moderation.Delegate) Handler {
	return Handler{authDelegate: authDelegate, delegate: delegate}
}

// InitRoutes mounts /api/users/:lmsUserId/ban* routes.
func (h *Handler) InitRoutes(router *gin.Engine) {
	group := router.Group("/api/users")
	group.GET("/:lmsUserId/ban", h.GetBanStatus)
	group.GET("/:lmsUserId/ban/history", h.GetBanHistory)
	group.POST("/:lmsUserId/ban", h.BanUser)
	group.POST("/:lmsUserId/unban", h.UnbanUser)
}

func (h *Handler) requireSuperAdmin(c *gin.Context) (callerID string, ok bool) {
	callerID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", false
	}
	if err := h.authDelegate.UserAccess(role, []models.Role{models.SuperAdmin}, c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", false
	}
	return callerID, true
}

func banJSON(ban *models.UserBanCore) gin.H {
	if ban == nil {
		return nil
	}
	now := time.Now().UTC()
	out := gin.H{
		"id":                ban.ID,
		"lmsUserId":         ban.LmsUserID,
		"reason":            ban.Reason,
		"bannedByLmsUserId": ban.BannedByLmsUserID,
		"bannedAt":          ban.BannedAt.UTC().Format(time.RFC3339),
		"isPermanent":       ban.IsPermanent(),
		"isActive":          ban.IsActive(now),
		"revokedSessions":   ban.RevokedSessions,
	}
	if ban.ExpiresAt != nil {
		out["expiresAt"] = ban.ExpiresAt.UTC().Format(time.RFC3339)
	} else {
		out["expiresAt"] = nil
	}
	if ban.UnbannedAt != nil {
		out["unbannedAt"] = ban.UnbannedAt.UTC().Format(time.RFC3339)
	} else {
		out["unbannedAt"] = nil
	}
	if ban.UnbannedByLmsUserID != nil {
		out["unbannedByLmsUserId"] = *ban.UnbannedByLmsUserID
	} else {
		out["unbannedByLmsUserId"] = nil
	}
	if ban.UnbanReason != nil {
		out["unbanReason"] = *ban.UnbanReason
	} else {
		out["unbanReason"] = nil
	}
	return out
}

func mapBanErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, modusecase.ErrCannotBanSelf):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "CANNOT_BAN_SELF"})
	case errors.Is(err, modusecase.ErrAlreadyBanned):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "ALREADY_BANNED"})
	case errors.Is(err, modusecase.ErrNotBanned):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "NOT_BANNED"})
	case errors.Is(err, modusecase.ErrReasonRequired):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "REASON_REQUIRED"})
	case errors.Is(err, modusecase.ErrInvalidExpiresAt), errors.Is(err, modusecase.ErrExpiresAtTooFar):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_EXPIRES_AT"})
	case errors.Is(err, auth.ErrUserNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error(), "code": "USER_NOT_FOUND"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// GetBanStatus godoc: GET /api/users/:lmsUserId/ban
func (h *Handler) GetBanStatus(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	lmsUserID := strings.TrimSpace(c.Param("lmsUserId"))
	ban, err := h.delegate.GetActiveBan(lmsUserID)
	if err != nil {
		mapBanErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"isBanned":  ban != nil,
		"activeBan": banJSON(ban),
	})
}

// GetBanHistory godoc: GET /api/users/:lmsUserId/ban/history?limit=20
func (h *Handler) GetBanHistory(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	lmsUserID := strings.TrimSpace(c.Param("lmsUserId"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.delegate.ListBanHistory(lmsUserID, limit)
	if err != nil {
		mapBanErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, ban := range items {
		out = append(out, banJSON(ban))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

type banBody struct {
	Reason    string  `json:"reason"`
	ExpiresAt *string `json:"expiresAt"`
}

// BanUser godoc: POST /api/users/:lmsUserId/ban
func (h *Handler) BanUser(c *gin.Context) {
	callerID, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	lmsUserID := strings.TrimSpace(c.Param("lmsUserId"))
	var body banBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "expiresAt must be RFC3339", "code": "INVALID_EXPIRES_AT"})
			return
		}
		expiresAt = &t
	}
	ban, err := h.delegate.BanUser(callerID, lmsUserID, body.Reason, expiresAt)
	if err != nil {
		mapBanErr(c, err)
		return
	}
	c.JSON(http.StatusOK, banJSON(ban))
}

type unbanBody struct {
	Reason string `json:"reason"`
}

// UnbanUser godoc: POST /api/users/:lmsUserId/unban
func (h *Handler) UnbanUser(c *gin.Context) {
	callerID, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	lmsUserID := strings.TrimSpace(c.Param("lmsUserId"))
	var body unbanBody
	_ = c.ShouldBindJSON(&body)
	ban, err := h.delegate.UnbanUser(callerID, lmsUserID, body.Reason)
	if err != nil {
		mapBanErr(c, err)
		return
	}
	c.JSON(http.StatusOK, banJSON(ban))
}
