package http

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/adminaudit"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	acusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	ufhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/http"
)

type Handler struct {
	authDelegate auth.Delegate
	uc           *acusecase.AdminContentUseCase
}

func NewAdminContentHandler(authDelegate auth.Delegate, uc *acusecase.AdminContentUseCase) Handler {
	return Handler{authDelegate: authDelegate, uc: uc}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	api := router.Group("/api/admin")
	{
		api.GET("/users/:userId/content", h.UserContent)
		api.GET("/users/:userId/audit", h.Audit)
		api.POST("/users/:userId/projects/:projectId/download", h.DownloadProject)
		api.POST("/users/:userId/files/:fileId/download", h.DownloadFile)
		api.DELETE("/users/:userId/projects/:projectId", h.DeleteProject)
		api.DELETE("/users/:userId/files/:fileId", h.DeleteFile)
		api.GET("/storage", h.Storage)
	}
}

func (h *Handler) requireSuperAdmin(c *gin.Context) (string, bool) {
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

func errorHandling(err error, c *gin.Context) {
	switch {
	case errors.Is(err, adminaudit.ErrReasonRequired):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "REASON_REQUIRED", "code": "REASON_REQUIRED"})
	case errors.Is(err, admincontent.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "NOT_FOUND", "code": "NOT_FOUND"})
	default:
		log.Printf("admincontent: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

// reason reads {reason}; a missing or broken body is the same as an empty reason.
func reason(c *gin.Context) string {
	var req reasonRequest
	_ = c.ShouldBindJSON(&req)
	return req.Reason
}

func (h *Handler) UserContent(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	content, err := h.uc.UserContent(c.Param("userId"))
	if err != nil {
		errorHandling(err, c)
		return
	}
	projects := make([]gin.H, 0, len(content.Projects))
	for _, p := range content.Projects {
		projects = append(projects, gin.H{
			"id":        p.ID,
			"title":     p.Title,
			"isPublic":  p.IsPublic,
			"sizeBytes": p.SizeBytes,
			"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	files := make([]models.UserFileHTTP, 0, len(content.Files))
	for _, f := range content.Files {
		files = append(files, f.ToHTTP())
	}
	c.JSON(http.StatusOK, gin.H{
		"usage": gin.H{
			"usedBytes":         content.Usage.Total(),
			"usedBytesProjects": content.Usage.Projects,
			"usedBytesFiles":    content.Usage.Files,
			"storageQuotaMb":    content.QuotaBytes >> 20,
		},
		"filesEnabled": content.FilesEnabled,
		"projects":     projects,
		"files":        files,
	})
}

func (h *Handler) Audit(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.uc.Audit(c.Param("userId"), limit)
	if err != nil {
		errorHandling(err, c)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		out = append(out, gin.H{
			"id":          it.ID,
			"actorUserId": it.ActorUserID,
			"actorName":   it.ActorName,
			"action":      it.Action,
			"objectType":  it.ObjectType,
			"objectId":    it.ObjectID,
			"objectName":  it.ObjectName,
			"sizeBytes":   it.SizeBytes,
			"reason":      it.Reason,
			"createdAt":   it.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) DownloadProject(c *gin.Context) {
	actor, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	data, filename, err := h.uc.DownloadProject(actor, c.Param("userId"), c.Param("projectId"), reason(c))
	if err != nil {
		errorHandling(err, c)
		return
	}
	ufhttp.SetDownloadHeaders(c.Writer, "application/x.scratch.sb3", filename, "")
	c.Data(http.StatusOK, "application/x.scratch.sb3", data)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	actor, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	row, rc, err := h.uc.DownloadFile(actor, c.Param("userId"), c.Param("fileId"), reason(c))
	if err != nil {
		errorHandling(err, c)
		return
	}
	defer rc.Close()
	ufhttp.ServeFile(c, row, rc)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	actor, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	if err := h.uc.DeleteProject(actor, c.Param("userId"), c.Param("projectId"), reason(c)); err != nil {
		errorHandling(err, c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteFile(c *gin.Context) {
	actor, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	if err := h.uc.DeleteFile(actor, c.Param("userId"), c.Param("fileId"), reason(c)); err != nil {
		errorHandling(err, c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Storage(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	report, err := h.uc.Storage()
	if err != nil {
		errorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, report)
}
