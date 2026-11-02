package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits"
	"log"
	"net/http"
)

type Handler struct {
	authDelegate       auth.Delegate
	robboUnitsDelegate robboUnits.Delegate
}

func NewRobboUnitsHandler(
	authDelegate auth.Delegate,
	robboUnits robboUnits.Delegate,
) Handler {
	return Handler{
		authDelegate:       authDelegate,
		robboUnitsDelegate: robboUnits,
	}
}

func (h *Handler) InitRobboUnitsRoutes(router *gin.Engine) {
	robboUnits := router.Group("/robboUnits")
	{
		robboUnits.POST("/", h.CreateRobboUnit)
		robboUnits.GET("/:robboUnitId", h.GetRobboUnitById)
		robboUnits.GET("/unitAdmin", h.GetRobboUnitsByUnitAdminId)
		robboUnits.GET("/", h.GetAllRobboUnits)
		robboUnits.PUT("/", h.UpdateRobboUnit)
		robboUnits.DELETE("/:robboUnitId", h.DeleteRobboUnit)
	}
}

type createRobboUnitResponse struct {
	RobboUnit *models.RobboUnitHTTP
}

func (h *Handler) CreateRobboUnit(c *gin.Context) {
	log.Println("Create Robbo Unit")
	_, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	robboUnitHttp := models.RobboUnitHTTP{}
	if err := c.BindJSON(&robboUnitHttp); err != nil {
		err = robboUnits.ErrBadRequestBody
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	robboUnit, err := h.robboUnitsDelegate.CreateRobboUnit(&robboUnitHttp)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, createRobboUnitResponse{
		&robboUnit,
	})
}

func (h *Handler) GetRobboUnitById(c *gin.Context) {
	log.Println("Get RobboUnit By Id")
	_, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := []models.Role{models.Teacher, models.UnitAdmin, models.SuperAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}
	robboUnitId := c.Param("robboUnitId")
	robboUnit, err := h.robboUnitsDelegate.GetRobboUnitById(robboUnitId)

	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, robboUnit)
}

func (h *Handler) GetAllRobboUnits(c *gin.Context) {
	log.Println("Get All RobboUnits")
	_, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}
	robboUnits, _, err := h.robboUnitsDelegate.GetAllRobboUnits("0", "0")
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, robboUnits)
}

func (h *Handler) GetRobboUnitsByUnitAdminId(c *gin.Context) {
	log.Println("Get RobboUnits By UnitAdminId")

	id, role, identityErr := h.authDelegate.UserIdentity(c)
	if identityErr != nil {
		log.Println(identityErr)
		ErrorHandling(identityErr, c)
		return
	}
	allowedRoles := []models.Role{models.UnitAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	robboUnits, _, err := h.robboUnitsDelegate.GetRobboUnitsByUnitAdminId(id, "0", "0")
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, robboUnits)
}

func (h *Handler) UpdateRobboUnit(c *gin.Context) {
	log.Println("Update RobboUnit")
	_, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}

	robboUnitHttp := models.RobboUnitHTTP{}

	if err := c.BindJSON(&robboUnitHttp); err != nil {
		err = robboUnits.ErrBadRequestBody
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	_, err := h.robboUnitsDelegate.UpdateRobboUnit(&robboUnitHttp)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) DeleteRobboUnit(c *gin.Context) {
	log.Println("Delete RobboUnit")
	_, role, userIdentityErr := h.authDelegate.UserIdentity(c)
	if userIdentityErr != nil {
		log.Println(userIdentityErr)
		ErrorHandling(userIdentityErr, c)
		return
	}
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := h.authDelegate.UserAccess(role, allowedRoles, c)
	if accessErr != nil {
		log.Println(accessErr)
		ErrorHandling(accessErr, c)
		return
	}
	robboUnitId := c.Param("robboUnitId")
	err := h.robboUnitsDelegate.DeleteRobboUnit(robboUnitId)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
	c.Status(http.StatusOK)
}

func ErrorHandling(err error, c *gin.Context) {
	switch {
	case errors.Is(err, robboUnits.ErrBadRequest):
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case errors.Is(err, robboUnits.ErrInternalServerLevel):
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	case errors.Is(err, robboUnits.ErrBadRequestBody):
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case errors.Is(err, robboUnits.ErrRobboUnitNotFound):
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	case errors.Is(err, auth.ErrInvalidAccessToken):
		c.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrTokenNotFound):
		c.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrNotAccess):
		c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	}
}
