package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
)

type Handler struct {
	authDelegate        auth.Delegate
	projectsDelegate    projects.Delegate
	projectPageDelegate projectPage.Delegate
}

func NewProjectsHandler(
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
	project := router.Group("/project")
	{
		project.POST("/", h.CreateProject)
		project.GET("/:projectId", h.GetProject)
		project.POST("/:projectId", h.UpdateProject)
		project.DELETE("/", h.DeleteProject)
	}
}

type testResponse struct {
	Id string `json:"id"`
}

func (h *Handler) CreateProject(c *gin.Context) {
	userId, userRole, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, userIdentityErr.Error())
		return
	}
	allowedRoles := []models.Role{models.Student}
	accessErr := h.authDelegate.UserAccess(userRole, allowedRoles, c)
	if accessErr != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, accessErr.Error())
		return
	}

	jsonDataBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	projectHTTP := models.ProjectHTTP{}
	projectHTTP.Json = string(jsonDataBytes)
	projectHTTP.AuthorId = userId

	projectId, err := h.projectsDelegate.CreateProject(&projectHTTP)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, testResponse{Id: projectId})
}

func (h *Handler) GetProject(c *gin.Context) {
	projectId := c.Param("projectId")
	if projectId == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if playToken := strings.TrimSpace(c.Query("token")); playToken != "" {
		jsonStr, err := h.projectPageDelegate.GetProjectJSONByToken(projectId, playToken)
		if err != nil {
			status := http.StatusInternalServerError
			if err == auth.ErrInvalidAccessToken {
				status = http.StatusUnauthorized
			}
			c.AbortWithStatusJSON(status, err.Error())
			return
		}
		var jsonMap map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, jsonMap)
		return
	}

	userId, userRole, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, userIdentityErr.Error())
		return
	}
	allowedRoles := []models.Role{
		models.Student, models.Teacher, models.Parent, models.FreeListener,
		models.UnitAdmin, models.SuperAdmin,
	}
	accessErr := h.authDelegate.UserAccess(userRole, allowedRoles, c)
	if accessErr != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, accessErr.Error())
		return
	}

	project, err := h.projectsDelegate.GetProjectById(projectId, userId)
	if err != nil {
		if err == auth.ErrNotAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(project.Json), &jsonMap)
	c.JSON(http.StatusOK, jsonMap)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	userId, userRole, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, userIdentityErr.Error())
		return
	}
	allowedRoles := []models.Role{models.Student}
	accessErr := h.authDelegate.UserAccess(userRole, allowedRoles, c)
	if accessErr != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, accessErr.Error())
		return
	}

	jsonDataBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	projectId := c.Param("projectId")
	projectHTTP := models.ProjectHTTP{}
	projectHTTP.ID = projectId
	projectHTTP.AuthorId = userId
	projectHTTP.Json = string(jsonDataBytes)

	err = h.projectsDelegate.UpdateProject(&projectHTTP, userId)
	if err != nil {
		if err == auth.ErrNotAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, testResponse{Id: projectId})
}

func (h *Handler) DeleteProject(c *gin.Context) {
}
