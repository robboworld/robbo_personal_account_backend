package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

type Handler struct {
	authDelegate auth.Delegate
	uc           teacherclass.UseCase
}

func NewTeacherClassHandler(authDelegate auth.Delegate, uc teacherclass.UseCase) Handler {
	return Handler{authDelegate: authDelegate, uc: uc}
}

func teacherRoles() []models.Role {
	return []models.Role{models.Teacher, models.SuperAdmin}
}

func studentRoles() []models.Role {
	return []models.Role{models.Student, models.Teacher, models.Parent, models.FreeListener, models.UnitAdmin, models.SuperAdmin}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	join := router.Group("/api/join")
	{
		join.GET("/preview", h.PreviewJoin)
		join.GET("/:slug/preview", h.PreviewJoinSlug)
		join.POST("", h.JoinClass)
	}

	api := router.Group("/api/teacher")
	{
		api.GET("/classes", h.ListMyClasses)
		api.POST("/classes", h.CreateClass)
		api.GET("/classes/:classId", h.GetClass)
		api.PATCH("/classes/:classId", h.RenameClass)
		api.DELETE("/classes/:classId", h.ArchiveClass)
		api.POST("/classes/:classId/rotate-invite", h.RotateInvite)
		api.GET("/classes/:classId/members", h.ListMembers)
		api.POST("/classes/:classId/members", h.AddMembers)
		api.DELETE("/classes/:classId/members/:username", h.RemoveMember)
		api.GET("/classes/:classId/assignments", h.ListAssignmentsTeacher)
		api.POST("/classes/:classId/assignments", h.CreateAssignment)
		api.POST("/classes/:classId/assignments/:assignmentId/issue-late", h.IssueLate)
		api.GET("/classes/:classId/assignments/:assignmentId/submissions", h.ListSubmissions)
		api.GET("/classes/:classId/live", h.LiveRoster)
		api.GET("/classes/:classId/progress", h.Progress)
		api.POST("/submissions/:projectId/review", h.ReviewSubmission)
	}

	student := router.Group("/api/student")
	{
		student.GET("/classes", h.ListStudentClasses)
		student.GET("/classes/:classId/assignments", h.ListAssignmentsStudent)
		student.POST("/projects/:projectId/submit", h.SubmitProject)
	}
}

func (h *Handler) identity(c *gin.Context, roles []models.Role) (userID string, role models.Role, ok bool) {
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", 0, false
	}
	if err := h.authDelegate.UserAccess(role, roles, c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", 0, false
	}
	return userID, role, true
}

func writeErr(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, teacherclass.ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, teacherclass.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, teacherclass.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, teacherclass.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, teacherclass.ErrConflict), errors.Is(err, teacherclass.ErrAlreadyIn):
		status = http.StatusConflict
	}
	c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
}

func (h *Handler) PreviewJoin(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		code = c.Query("slug")
	}
	dto, err := h.uc.PreviewJoin(code)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) PreviewJoinSlug(c *gin.Context) {
	dto, err := h.uc.PreviewJoin(c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) JoinClass(c *gin.Context) {
	userID, _, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	var body struct {
		Code string `json:"code"`
		Slug string `json:"slug"`
	}
	_ = c.ShouldBindJSON(&body)
	key := body.Slug
	if key == "" {
		key = body.Code
	}
	dto, err := h.uc.JoinClass(userID, key)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) ListMyClasses(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListMyClasses(userID)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) CreateClass(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		CourseID    string `json:"courseId"`
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	dto, err := h.uc.CreateClass(userID, teacherclass.CreateClassInput{
		CourseID:    body.CourseID,
		DisplayName: body.DisplayName,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto)
}

func (h *Handler) GetClass(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	dto, err := h.uc.GetClass(userID, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) RenameClass(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	dto, err := h.uc.RenameClass(userID, c.Param("classId"), body.DisplayName)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) ArchiveClass(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	if err := h.uc.ArchiveClass(userID, c.Param("classId")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RotateInvite(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	dto, err := h.uc.RotateInvite(userID, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) ListMembers(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListMembers(userID, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) AddMembers(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		Emails []string `json:"emails"`
		Text   string   `json:"text"`
	}
	_ = c.ShouldBindJSON(&body)
	emails := body.Emails
	if body.Text != "" {
		for _, part := range strings.FieldsFunc(body.Text, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' '
		}) {
			if part != "" {
				emails = append(emails, part)
			}
		}
	}
	added, missing, err := h.uc.AddMembersByEmail(userID, c.Param("classId"), emails)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"added": added, "missing": missing})
}

func (h *Handler) RemoveMember(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	if err := h.uc.RemoveMember(userID, c.Param("classId"), c.Param("username")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateAssignment(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		Title             string  `json:"title"`
		Instructions      string  `json:"instructions"`
		AssignmentKey     string  `json:"assignmentKey"`
		TemplateProjectID string  `json:"templateProjectId"`
		DueAt             *string `json:"dueAt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	var due *time.Time
	if body.DueAt != nil && *body.DueAt != "" {
		t, err := time.Parse(time.RFC3339, *body.DueAt)
		if err != nil {
			writeErr(c, teacherclass.ErrBadRequest)
			return
		}
		due = &t
	}
	key := body.AssignmentKey
	if key == "" {
		key = strings.ToLower(strings.ReplaceAll(body.Title, " ", "-"))
	}
	dto, err := h.uc.CreateAssignment(userID, c.Param("classId"), teacherclass.CreateAssignmentInput{
		Title:             body.Title,
		Instructions:      body.Instructions,
		AssignmentKey:     key,
		TemplateProjectID: body.TemplateProjectID,
		DueAt:             due,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto)
}

func (h *Handler) ListAssignmentsTeacher(c *gin.Context) {
	userID, role, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListAssignments(userID, role, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) IssueLate(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	n, err := h.uc.IssueToLatecomers(userID, c.Param("classId"), c.Param("assignmentId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"issued": n})
}

func (h *Handler) ListSubmissions(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListSubmissions(userID, c.Param("classId"), c.Param("assignmentId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) LiveRoster(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	list, err := h.uc.LiveRoster(userID, c.Param("classId"), c.Query("assignmentId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) Progress(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	assignments, rows, err := h.uc.ProgressMatrix(userID, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignments": assignments, "rows": rows})
}

func (h *Handler) ReviewSubmission(c *gin.Context) {
	userID, _, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	if err := h.uc.ReviewSubmission(userID, c.Param("projectId"), body.Status, body.Comment); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListStudentClasses(c *gin.Context) {
	userID, _, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListStudentClasses(userID)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) ListAssignmentsStudent(c *gin.Context) {
	userID, role, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	list, err := h.uc.ListAssignments(userID, role, c.Param("classId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) SubmitProject(c *gin.Context) {
	userID, _, ok := h.identity(c, []models.Role{models.Student, models.Teacher, models.SuperAdmin})
	if !ok {
		return
	}
	if err := h.uc.SubmitAssignment(userID, c.Param("projectId")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
