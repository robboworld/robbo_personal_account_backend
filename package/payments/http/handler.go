package http

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
)

type Handler struct {
	authDelegate     auth.Delegate
	paymentsDelegate payments.Delegate
}

func NewPaymentsHandler(authDelegate auth.Delegate, paymentsDelegate payments.Delegate) Handler {
	return Handler{
		authDelegate:     authDelegate,
		paymentsDelegate: paymentsDelegate,
	}
}

func (h *Handler) InitPaymentsRoutes(router *gin.Engine) {
	router.GET("/payments/products", h.ListProducts)
	router.POST("/payments/products", h.CreateProduct)
	router.POST("/payments/checkout", h.Checkout)
	router.GET("/payments/orders/:orderNumber", h.GetOrder)
	router.POST("/payments/webhook/yookassa", h.YookassaWebhook)
}

func (h *Handler) ListProducts(c *gin.Context) {
	list, err := h.paymentsDelegate.ListActiveProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, p := range list {
		out = append(out, productToJSON(p))
	}
	c.JSON(http.StatusOK, gin.H{"products": out})
}

type createProductRequest struct {
	SKU          string   `json:"sku"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Amount       float64  `json:"amount"`
	Currency     string   `json:"currency"`
	SeatLimit    int      `json:"seatLimit"`
	Capabilities []string `json:"capabilities"`
	DurationDays int      `json:"durationDays"`
	IsActive     *bool    `json:"isActive"`
}

func (h *Handler) CreateProduct(c *gin.Context) {
	userID, userRole, err := h.sessionIdentity(c)
	if err != nil || userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if accessErr := h.authDelegate.UserAccess(userRole, []models.Role{models.SuperAdmin}, c); accessErr != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var body createProductRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	product, err := h.paymentsDelegate.CreateProduct(models.CreateProductInput{
		SKU:          body.SKU,
		Title:        body.Title,
		Description:  body.Description,
		Amount:       body.Amount,
		Currency:     body.Currency,
		SeatLimit:    body.SeatLimit,
		Capabilities: body.Capabilities,
		DurationDays: body.DurationDays,
		IsActive:     active,
	})
	if err != nil {
		writePaymentsError(c, err)
		return
	}
	c.JSON(http.StatusOK, productToJSON(product))
}

type checkoutRequest struct {
	ProductID string `json:"productId"`
}

func (h *Handler) Checkout(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil || userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body checkoutRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.ProductID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	email := ""
	if raw, ok := c.Get("user_email"); ok {
		email, _ = raw.(string)
	}
	confirmationURL, order, err := h.paymentsDelegate.Checkout(userID, strings.TrimSpace(body.ProductID), email)
	if err != nil {
		writePaymentsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"confirmationUrl": confirmationURL,
		"orderNumber":     order.OrderNumber,
		"order":           orderToJSON(order),
	})
}

func (h *Handler) GetOrder(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil || userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orderNumber := c.Param("orderNumber")
	sourceIP := clientIP(c)
	order, err := h.paymentsDelegate.SyncOrder(userID, orderNumber, sourceIP)
	if err != nil {
		writePaymentsError(c, err)
		return
	}
	c.JSON(http.StatusOK, orderToJSON(order))
}

func (h *Handler) YookassaWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	err = h.paymentsDelegate.HandleWebhook(body, clientIP(c))
	if errors.Is(err, payments.ErrUntrustedWebhookSource) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "untrusted_source"})
		return
	}
	if errors.Is(err, payments.ErrBadRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	// Always 200 for recognized/handled events so YooKassa does not retry forever.
	c.Status(http.StatusOK)
}

func (h *Handler) sessionIdentity(c *gin.Context) (userID string, role models.Role, err error) {
	if rawID, ok := c.Get("user_id"); ok {
		id, _ := rawID.(string)
		id = strings.TrimSpace(id)
		if id != "" && id != "0" {
			role = models.Anonymous
			if rawRole, ok := c.Get("user_role"); ok {
				if r, ok := rawRole.(models.Role); ok {
					role = r
				}
			}
			if role != models.Anonymous && role != models.WithExpiredToken {
				return id, role, nil
			}
		}
	}
	return h.authDelegate.UserIdentity(c)
}

func clientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return ip
	}
	return c.ClientIP()
}

func writePaymentsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, payments.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "product_not_found", "errorCode": "PRODUCT_NOT_FOUND"})
	case errors.Is(err, payments.ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order_not_found", "errorCode": "ORDER_NOT_FOUND"})
	case errors.Is(err, payments.ErrPaymentNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment_not_configured", "errorCode": "PAYMENT_NOT_CONFIGURED"})
	case errors.Is(err, payments.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, payments.ErrYookassaPaymentFailed):
		c.JSON(http.StatusBadGateway, gin.H{"error": "yookassa_payment_failed", "errorCode": "YOOKASSA_PAYMENT_FAILED"})
	case errors.Is(err, payments.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func productToJSON(p *models.ProductCore) gin.H {
	return gin.H{
		"id":           p.ID,
		"sku":          p.SKU,
		"title":        p.Title,
		"description":  p.Description,
		"amount":       p.Amount,
		"currency":     p.Currency,
		"seatLimit":    p.SeatLimit,
		"capabilities": p.Capabilities,
		"durationDays": p.DurationDays,
		"isActive":     p.IsActive,
	}
}

func orderToJSON(o *models.OrderCore) gin.H {
	out := gin.H{
		"id":                o.ID,
		"orderNumber":       o.OrderNumber,
		"lmsUserId":         o.LmsUserID,
		"productId":         o.ProductID,
		"amount":            o.Amount,
		"currency":          o.Currency,
		"status":            o.Status,
		"yookassaPaymentId": o.YookassaPaymentID,
		"licenseId":         o.LicenseID,
		"createdAt":         o.CreatedAt.Format(time.RFC3339),
		"updatedAt":         o.UpdatedAt.Format(time.RFC3339),
	}
	if o.PaidAt != nil {
		out["paidAt"] = o.PaidAt.Format(time.RFC3339)
	}
	return out
}
