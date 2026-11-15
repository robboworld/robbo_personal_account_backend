package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
)

// Handler serves admin LMS user search endpoints.
type Handler struct {
	authDelegate auth.Delegate
	service      *usersearch.Service
}

// NewHandler creates a user search HTTP handler.
func NewHandler(authDelegate auth.Delegate, service *usersearch.Service) Handler {
	return Handler{authDelegate: authDelegate, service: service}
}

// InitRoutes mounts /api/users/search routes.
func (h *Handler) InitRoutes(router *gin.Engine) {
	group := router.Group("/api/users")
	group.GET("/search", h.Search)
	group.POST("/search/reindex", h.Reindex)
}

func (h *Handler) requireAdmin(c *gin.Context, roles []models.Role) (models.Role, bool) {
	_, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return role, false
	}
	if err := h.authDelegate.UserAccess(role, roles, c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return role, false
	}
	return role, true
}

// Search godoc: GET /api/users/search?q=&limit=20&beforeId=&includeInactive=true
// Without q a SuperAdmin gets every user newest first (the users page); the next page passes
// the last id shown as beforeId. A UnitAdmin still has to type a query.
func (h *Handler) Search(c *gin.Context) {
	role, ok := h.requireAdmin(c, []models.Role{models.UnitAdmin, models.SuperAdmin})
	if !ok {
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	includeInactive := role == models.SuperAdmin &&
		(strings.EqualFold(c.Query("includeInactive"), "true") || c.Query("includeInactive") == "1")
	if h.service == nil || (q == "" && role != models.SuperAdmin) {
		c.JSON(http.StatusOK, gin.H{"items": []usersearch.Hit{}})
		return
	}
	var items []usersearch.Hit
	var err error
	if q == "" {
		beforeID, _ := strconv.ParseInt(c.Query("beforeId"), 10, 64)
		items, err = h.service.List(limit, beforeID, includeInactive)
	} else {
		items, err = h.service.Search(c.Request.Context(), q, limit, includeInactive)
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []usersearch.Hit{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Reindex godoc: POST /api/users/search/reindex (SuperAdmin)
func (h *Handler) Reindex(c *gin.Context) {
	if _, ok := h.requireAdmin(c, []models.Role{models.SuperAdmin}); !ok {
		return
	}
	if h.service == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "search unavailable"})
		return
	}
	if err := h.service.EnsureIndex(c.Request.Context()); err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	n, err := h.service.ReindexAll(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "documents": n})
}
