package http

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type Handler struct {
	authDelegate         auth.Delegate
	coursePacketDelegate coursePacket.Delegate
}

func NewCoursePacketHandler(
	authDelegate auth.Delegate,
	coursePacketDelegate coursePacket.Delegate,
) Handler {
	return Handler{
		authDelegate:         authDelegate,
		coursePacketDelegate: coursePacketDelegate,
	}
}

type testCourseResponse struct {
	CourseId string `json:"courseId"`
}

func (h *Handler) InitCoursePacketRoutes(router *gin.Engine) {
	course := router.Group("/coursePacket")
	{
		course.POST("/:coursePacketId", h.CreateCoursePacket)
		course.GET("/:coursePacketId", h.GetCoursePacketById)
		course.GET("/", h.GetAllCoursePackets)
		course.PUT("/", h.UpdateCoursePacket)
		course.DELETE("/:coursePacketId", h.DeleteCoursePacket)
	}
}

func (h *Handler) UpdateCoursePacket(c *gin.Context) {
	log.Println("Update Course Packet")
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
	coursePacketHTTP := models.CoursePacketHTTP{}
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		err = coursePacket.ErrBadRequestBody
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	err = json.Unmarshal(body, &coursePacketHTTP)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	err = h.coursePacketDelegate.UpdateCoursePacket(&coursePacketHTTP)
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CreateCoursePacket(c *gin.Context) {
	log.Println("Create Course Packet")
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
	courseId := c.Param("coursePacketId")
	coursePacketHTTP := models.CoursePacketHTTP{}
	courseId, err := h.coursePacketDelegate.CreateCoursePacket(&coursePacketHTTP, courseId)

	if err != nil {
		log.Println(err)
		err = coursePacket.ErrBadRequest
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, testCourseResponse{
		courseId,
	})
}

func (h *Handler) GetCoursePacketById(c *gin.Context) {
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
	coursePacketId := c.Param("coursePacketId")

	crsPacket, err := h.coursePacketDelegate.GetCoursePacketById(coursePacketId)

	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, crsPacket)
}

func (h *Handler) GetAllCoursePackets(c *gin.Context) {
	log.Println("Get all CoursePackets")
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
	coursePackets, err := h.coursePacketDelegate.GetAllCoursePackets()
	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}

	c.JSON(http.StatusOK, coursePackets)
}

func (h *Handler) DeleteCoursePacket(c *gin.Context) {
	log.Println("Delete Course Packet")
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
	courseId := c.Param("coursePacketId")
	err := h.coursePacketDelegate.DeleteCoursePacket(courseId)

	if err != nil {
		log.Println(err)
		ErrorHandling(err, c)
		return
	}
}

func ErrorHandling(err error, c *gin.Context) {
	switch err {
	case coursePacket.ErrBadRequest:
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case coursePacket.ErrInternalServerLevel:
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	case coursePacket.ErrBadRequestBody:
		c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	case coursePacket.ErrCoursePacketNotFound:
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
