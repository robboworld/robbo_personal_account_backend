package http

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
)

func allAuthenticatedRoles() []models.Role {
	return []models.Role{
		models.Student,
		models.Teacher,
		models.Parent,
		models.FreeListener,
		models.UnitAdmin,
		models.SuperAdmin,
	}
}

func projectCreateRoles() []models.Role {
	return []models.Role{models.Student, models.UnitAdmin, models.SuperAdmin}
}

func backendBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if fwd := c.GetHeader("X-Forwarded-Proto"); fwd != "" {
		scheme = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host := c.Request.Host
	if host == "" {
		host = "localhost:8080"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

type Handler struct {
	authDelegate        auth.Delegate
	projectsDelegate    projects.Delegate
	projectPageDelegate projectPage.Delegate
}

func NewProjectPageHandler(
	authDelegate auth.Delegate,
	projectsDelegate projects.Delegate,
	projectPageDelegate projectPage.Delegate,
) Handler {
	return Handler{
		authDelegate:        authDelegate,
		projectsDelegate:    projectsDelegate,
		projectPageDelegate: projectPageDelegate,
	}
}

func (h *Handler) InitProjectRoutes(router *gin.Engine) {
	projectPageGroup := router.Group("/projectPage")
	{
		projectPageGroup.POST("/", h.CreateProjectPage)
		projectPageGroup.GET("/public", h.GetPublicProjectPages)
		projectPageGroup.GET("/reaction-types", h.ListReactionTypes)
		projectPageGroup.GET("/:projectPageId/reactions", h.GetProjectReactions)
		projectPageGroup.PUT("/:projectPageId/reactions", h.PutProjectReaction)
		projectPageGroup.DELETE("/:projectPageId/reactions", h.DeleteProjectReaction)
		projectPageGroup.GET("/:projectPageId/play-token", h.IssuePlayToken)
		projectPageGroup.GET("/:projectPageId/play", h.PlayProjectSb3)
		projectPageGroup.GET("/:projectPageId/download", h.DownloadProjectSb3)
		projectPageGroup.POST("/:projectPageId/upload", h.UploadProjectSb3)
		projectPageGroup.GET("/:projectPageId/preview", h.GetProjectPreview)
		projectPageGroup.POST("/:projectPageId/preview", h.UploadProjectPreview)
		projectPageGroup.GET("/:projectPageId", h.GetProjectPageById)
		projectPageGroup.GET("/", h.GetAllProjectPageByUserId)
		projectPageGroup.PUT("/", h.UpdateProjectPage)
		projectPageGroup.DELETE("/:projectPageId", h.DeleteProjectPage)
	}
}

// optionalViewerID returns authenticated user id when present; empty string for guests.
func optionalViewerID(c *gin.Context, authDelegate auth.Delegate) string {
	if id, _, err := authDelegate.UserIdentity(c); err == nil {
		id = strings.TrimSpace(id)
		if id != "" && id != "0" {
			return id
		}
	}
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" && s != "0" {
				return s
			}
		}
	}
	return ""
}

func (h *Handler) ListReactionTypes(c *gin.Context) {
	types, err := h.projectPageDelegate.ListEnabledReactionTypes()
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	if types == nil {
		types = []models.ReactionTypeHTTP{}
	}
	c.JSON(http.StatusOK, gin.H{"types": types})
}

func (h *Handler) GetProjectReactions(c *gin.Context) {
	viewerID := ""
	if id, _, err := h.authDelegate.UserIdentity(c); err == nil {
		viewerID = id
	}
	summary, err := h.projectPageDelegate.GetProjectReactions(c.Param("projectPageId"), viewerID)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) reactionIdentity(c *gin.Context) (string, bool) {
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", false
	}
	if err := h.authDelegate.UserAccess(role, allAuthenticatedRoles(), c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", false
	}
	return userID, true
}

func (h *Handler) PutProjectReaction(c *gin.Context) {
	userID, ok := h.reactionIdentity(c)
	if !ok {
		return
	}
	var request struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		ErrorHandling(projectPage.ErrBadRequestBody, c)
		return
	}
	summary, err := h.projectPageDelegate.PutProjectReaction(
		c.Param("projectPageId"),
		userID,
		request.Code,
	)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) DeleteProjectReaction(c *gin.Context) {
	userID, ok := h.reactionIdentity(c)
	if !ok {
		return
	}
	summary, err := h.projectPageDelegate.DeleteProjectReaction(c.Param("projectPageId"), userID)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, summary)
}

type createProjectPageResponse struct {
	ProjectPage *models.ProjectPageHTTP `json:"projectPage"`
}

func (h *Handler) CreateProjectPage(c *gin.Context) {
	log.Println("Create Project Page")
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := projectCreateRoles()
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	projectPage, err := h.projectPageDelegate.CreateProjectPage(userId)

	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, createProjectPageResponse{
		&projectPage,
	})
}

type getProjectPageResponse struct {
	ProjectPage *models.ProjectPageHTTP    `json:"projectPage"`
	PlayToken   *projectPage.PlayTokenHTTP `json:"playToken,omitempty"`
}

func (h *Handler) GetProjectPageById(c *gin.Context) {
	log.Println("Get Project Page By ID")
	userId := optionalViewerID(c, h.authDelegate)
	projectPageId := c.Param("projectPageId")
	page, err := h.projectPageDelegate.GetProjectPageById(projectPageId, userId)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	var playToken *projectPage.PlayTokenHTTP
	if token, tokenErr := h.projectPageDelegate.IssuePlayToken(projectPageId, userId, backendBaseURL(c)); tokenErr == nil {
		playToken = token
	}

	c.JSON(http.StatusOK, getProjectPageResponse{
		ProjectPage: &page,
		PlayToken:   playToken,
	})
}

func asciiAttachmentFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" || s == ".sb3" || !strings.HasSuffix(strings.ToLower(s), ".sb3") {
		return "scratch-project.sb3"
	}
	return s
}

func (h *Handler) DownloadProjectSb3(c *gin.Context) {
	projectPageId := c.Param("projectPageId")
	var data []byte
	var filename string
	var err error
	if playToken := strings.TrimSpace(c.Query("token")); playToken != "" {
		data, filename, err = h.projectPageDelegate.PlayProjectSb3ByToken(projectPageId, playToken)
	} else {
		userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
		if userIdentityErr != nil {
			log.Println(userIdentityErr)
			ErrorHandling(userIdentityErr, c)
			return
		}
		allowedRoles := allAuthenticatedRoles()
		accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
		if accessErr != nil {
			log.Println(accessErr)
			ErrorHandling(accessErr, c)
			return
		}
		data, filename, err = h.projectPageDelegate.DownloadProjectSb3(projectPageId, userId)
	}
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	cd := fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`,
		asciiAttachmentFilename(filename),
		url.PathEscape(filename),
	)
	c.Header("Content-Disposition", cd)
	c.Header("Content-Type", "application/zip")
	c.Data(http.StatusOK, "application/zip", data)
}

func (h *Handler) UploadProjectSb3(c *gin.Context) {
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		ErrorHandling(userIdentityErr, c)
		return
	}
	if accessErr := h.authDelegate.UserAccess(role, projectCreateRoles(), c); accessErr != nil {
		ErrorHandling(accessErr, c)
		return
	}
	projectPageId := c.Param("projectPageId")
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid multipart form")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "file is required")
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".sb3") {
		c.AbortWithStatusJSON(http.StatusBadRequest, "only .sb3 files are supported")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	if err := h.projectPageDelegate.UploadProjectSb3(projectPageId, userId, data); err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type getAllProjectPageResponse struct {
	ProjectPages []*models.ProjectPageHTTP `json:"projectPages"`
}

func (h *Handler) GetAllProjectPageByUserId(c *gin.Context) {
	log.Println("Get All Project Page By User ID")
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := projectCreateRoles()
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	projectPages, _, err := h.projectPageDelegate.GetAllProjectPagesByUserId(userId, "1", "10")
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, getAllProjectPageResponse{
		ProjectPages: projectPages,
	})
}

type updateProjectPageInput struct {
	ProjectPage *models.ProjectPageHTTP `json:"projectPage"`
}

func (h *Handler) UpdateProjectPage(c *gin.Context) {
	log.Println("Update Project Page")
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := projectCreateRoles()
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	inp := new(updateProjectPageInput)
	if err := c.BindJSON(&inp); err != nil {
		err = projectPage.ErrBadRequestBody
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	log.Println(inp)
	_, err := h.projectPageDelegate.UpdateProjectPage(inp.ProjectPage, userId)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteProjectPage(c *gin.Context) {
	log.Println("Delete Project Page")
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := projectCreateRoles()
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}
	projectId := c.Param("projectPageId")

	err := h.projectPageDelegate.DeleteProjectPage(projectId, userId)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	c.Status(http.StatusOK)
}

type playTokenResponse struct {
	PlayToken *projectPage.PlayTokenHTTP `json:"playToken"`
}

func (h *Handler) IssuePlayToken(c *gin.Context) {
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		ErrorHandling(userIdentityErr, c)
		return
	}
	if accessErr := h.authDelegate.UserAccess(role, allAuthenticatedRoles(), c); accessErr != nil {
		ErrorHandling(accessErr, c)
		return
	}
	projectPageId := c.Param("projectPageId")
	token, err := h.projectPageDelegate.IssuePlayToken(projectPageId, userId, backendBaseURL(c))
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, playTokenResponse{PlayToken: token})
}

func (h *Handler) PlayProjectSb3(c *gin.Context) {
	projectPageId := c.Param("projectPageId")
	if playToken := strings.TrimSpace(c.Query("token")); playToken != "" {
		data, _, err := h.projectPageDelegate.PlayProjectSb3ByToken(projectPageId, playToken)
		if err != nil {
			ErrorHandling(err, c)
			return
		}
		c.Header("Content-Disposition", "inline")
		c.Header("Content-Type", "application/zip")
		c.Data(http.StatusOK, "application/zip", data)
		return
	}
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		ErrorHandling(userIdentityErr, c)
		return
	}
	if accessErr := h.authDelegate.UserAccess(role, allAuthenticatedRoles(), c); accessErr != nil {
		ErrorHandling(accessErr, c)
		return
	}
	data, _, err := h.projectPageDelegate.PlayProjectSb3(projectPageId, userId)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.Header("Content-Disposition", "inline")
	c.Header("Content-Type", "application/zip")
	c.Data(http.StatusOK, "application/zip", data)
}

type getPublicProjectPagesResponse struct {
	ProjectPages []*models.ProjectPageHTTP `json:"projectPages"`
	CountRows    int                       `json:"countRows"`
}

func (h *Handler) GetPublicProjectPages(c *gin.Context) {
	// Public catalog is guest-readable; do not require Authorization.
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "10")
	featured := strings.ToLower(strings.TrimSpace(c.Query("featured")))
	landingFeaturedOnly := featured == "landing" || featured == "olympiad" || featured == "1" || featured == "true"
	projectPages, countRows, err := h.projectPageDelegate.GetPublicProjectPages(page, pageSize, landingFeaturedOnly)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, getPublicProjectPagesResponse{
		ProjectPages: projectPages,
		CountRows:    countRows,
	})
}

const maxPreviewUploadBytes = 2 * 1024 * 1024

func detectPreviewMime(data []byte, filename string) (string, bool) {
	mime := http.DetectContentType(data)
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
		return mime, true
	}
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg", true
	case strings.HasSuffix(lower, ".png"):
		return "image/png", true
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp", true
	}
	return "", false
}

func (h *Handler) GetProjectPreview(c *gin.Context) {
	userId := optionalViewerID(c, h.authDelegate)
	projectPageId := c.Param("projectPageId")
	data, mime, err := h.projectPageDelegate.GetPreviewImage(projectPageId, userId)
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, mime, data)
}

func (h *Handler) UploadProjectPreview(c *gin.Context) {
	userId, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		ErrorHandling(userIdentityErr, c)
		return
	}
	if accessErr := h.authDelegate.UserAccess(role, projectCreateRoles(), c); accessErr != nil {
		ErrorHandling(accessErr, c)
		return
	}
	projectPageId := c.Param("projectPageId")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		ErrorHandling(projectPage.ErrBadRequestBody, c)
		return
	}
	if fileHeader.Size > maxPreviewUploadBytes {
		c.AbortWithStatusJSON(http.StatusBadRequest, "preview image too large (max 2MB)")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxPreviewUploadBytes+1))
	if err != nil {
		ErrorHandling(err, c)
		return
	}
	if len(data) == 0 || len(data) > maxPreviewUploadBytes {
		c.AbortWithStatusJSON(http.StatusBadRequest, "preview image too large (max 2MB)")
		return
	}
	mime, ok := detectPreviewMime(data, fileHeader.Filename)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, "only jpeg/png/webp preview images are supported")
		return
	}
	if err := h.projectPageDelegate.SavePreviewImage(projectPageId, userId, data, mime); err != nil {
		ErrorHandling(err, c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"preview": "/projectPage/" + projectPageId + "/preview",
	})
}

func ErrorHandling(err error, c *gin.Context) {
	switch err {
	case projectPage.ErrBadRequest:
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case projectPage.ErrInternalServerLevel:
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	case projectPage.ErrPageNotFound:
		c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
	case projectPage.ErrSb3ArchiveNotFound:
		c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
	case projectPage.ErrBadRequestBody:
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case projects.ErrProjectNotFound:
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	case auth.ErrInvalidAccessToken:
		c.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
	case auth.ErrTokenNotFound:
		c.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
	case auth.ErrNotAccess:
		c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	}
}
