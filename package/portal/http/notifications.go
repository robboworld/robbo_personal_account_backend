package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/spf13/viper"
)

type NotificationsHandler struct {
	portal portalgateway.Gateway
}

func NewNotificationsHandler(portal portalgateway.Gateway) NotificationsHandler {
	return NotificationsHandler{portal: portal}
}

func (h NotificationsHandler) InitRoutes(router *gin.Engine) {
	router.POST("/internal/lms/notifications", h.LMSIngest)
}

type lmsNotificationIngest struct {
	RecipientUserID string `json:"recipientUserId"`
	RecipientEmail  string `json:"recipientEmail"`
	Title           string `json:"title"`
	Body            string `json:"body"`
	Kind            string `json:"kind"`
	Severity        string `json:"severity"`
	ActionURL       string `json:"actionUrl"`
	DedupeKey       string `json:"dedupeKey"`
}

func (h NotificationsHandler) LMSIngest(c *gin.Context) {
	if !viper.GetBool("lmsNotifications.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "ingest disabled"})
		return
	}
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if token == "" || token != viper.GetString("lmsNotifications.ingestBearerToken") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body lmsNotificationIngest
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if body.RecipientUserID == "" && body.RecipientEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient required"})
		return
	}
	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	severity := strings.ToUpper(body.Severity)
	if severity == "" {
		severity = "INFO"
	}
	n := &models.RobboPortalNotificationDB{
		Source:   "lms",
		Severity: severity,
		Title:    body.Title,
		Body:     body.Body,
	}
	if body.Kind != "" {
		n.Kind = &body.Kind
	}
	if body.ActionURL != "" {
		n.ActionURL = &body.ActionURL
	}
	if body.DedupeKey != "" {
		n.DedupeKey = &body.DedupeKey
	}
	if body.RecipientEmail != "" {
		email := body.RecipientEmail
		n.RecipientEmail = &email
	}
	if body.RecipientUserID != "" {
		if link, err := h.portal.FindUserLinkByLegacyOrEdx(body.RecipientUserID); err == nil && link != nil {
			n.RecipientUserLinkID = &link.ID
		}
	}
	dup, err := h.portal.CreateNotification(n)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dup {
		c.JSON(http.StatusOK, gin.H{"status": "duplicate"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}
