package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
)

type Handler struct {
	authDelegate      auth.Delegate
	licensingDelegate licensing.Delegate
}

func NewLicensingHandler(authDelegate auth.Delegate, licensingDelegate licensing.Delegate) Handler {
	return Handler{
		authDelegate:      authDelegate,
		licensingDelegate: licensingDelegate,
	}
}

func (h *Handler) InitLicensingRoutes(router *gin.Engine) {
	router.OPTIONS("/v1/activate", h.corsPreflight)
	router.OPTIONS("/v1/seats/deactivate", h.corsPreflight)
	router.OPTIONS("/v1/device/link/start", h.corsPreflight)
	router.OPTIONS("/v1/device/link/poll", h.corsPreflight)
	router.OPTIONS("/addon/manifest.json", h.corsPreflight)
	router.OPTIONS("/addon/paid-addon.js.enc", h.corsPreflight)
	router.OPTIONS("/health/licensing", h.corsPreflight)

	router.POST("/v1/activate", h.Activate)
	router.POST("/v1/seats/deactivate", h.DeactivateSeat)
	router.POST("/v1/device/link/start", h.StartDeviceLink)
	router.POST("/v1/device/link/poll", h.PollDeviceLink)
	router.GET("/addon/manifest.json", h.AddonManifest)
	router.GET("/addon/paid-addon.js.enc", h.AddonBundle)
	router.GET("/health/licensing", h.Health)

	api := router.Group("/licensing")
	{
		api.POST("/issue", h.IssueLicense)
		api.GET("/mine", h.ListMyLicenses)
		api.GET("/:licenseId", h.GetLicense)
		api.DELETE("/:licenseId/seats/:seatId", h.RevokeSeat)
		api.POST("/device/link/confirm", h.ConfirmDeviceLink)
	}
}

func (h *Handler) corsPreflight(c *gin.Context) {
	setCORS(c)
	c.Status(http.StatusNoContent)
}

func setCORS(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-RS3-Device-Fingerprint")
}

func (h *Handler) Health(c *gin.Context) {
	setCORS(c)
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"service": "licensing",
	})
}

type activateRequest struct {
	LicenseKey       string `json:"licenseKey"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	DeviceID         string `json:"deviceId"`
	PublicBase       string `json:"publicBase"`
	Device           *struct {
		Fingerprint string `json:"fingerprint"`
	} `json:"device"`
}

type deactivateRequest struct {
	LicenseKey        string `json:"licenseKey"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	Device            *struct {
		Fingerprint string `json:"fingerprint"`
	} `json:"device"`
}

func extractFingerprint(c *gin.Context, body activateRequest) string {
	if h := strings.TrimSpace(c.GetHeader("X-RS3-Device-Fingerprint")); h != "" {
		return h
	}
	if body.Device != nil && strings.TrimSpace(body.Device.Fingerprint) != "" {
		return strings.TrimSpace(body.Device.Fingerprint)
	}
	if strings.TrimSpace(body.DeviceFingerprint) != "" {
		return strings.TrimSpace(body.DeviceFingerprint)
	}
	if len(strings.TrimSpace(body.DeviceID)) >= 32 {
		return strings.TrimSpace(body.DeviceID)
	}
	return ""
}

func publicBaseFromRequest(c *gin.Context, bodyPublicBase string) string {
	if strings.TrimSpace(bodyPublicBase) != "" {
		return strings.TrimRight(strings.TrimSpace(bodyPublicBase), "/")
	}
	proto := c.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return "http://localhost:8080"
	}
	return proto + "://" + host
}

func (h *Handler) Activate(c *gin.Context) {
	setCORS(c)
	var body activateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	fp := extractFingerprint(c, body)
	result, err := h.licensingDelegate.Activate(body.LicenseKey, fp, publicBaseFromRequest(c, body.PublicBase))
	if err != nil {
		writeActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"signedOfflineToken": result.SignedOfflineToken,
		"addonManifestUrl":   result.AddonManifestUrl,
		"serverTime":         result.ServerTime.Format(time.RFC3339),
		"refreshAfter":       result.RefreshAfter.Format(time.RFC3339),
		"seatId":             result.SeatID,
	})
}

func (h *Handler) DeactivateSeat(c *gin.Context) {
	setCORS(c)
	var body deactivateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	fp := ""
	if h := strings.TrimSpace(c.GetHeader("X-RS3-Device-Fingerprint")); h != "" {
		fp = h
	} else if body.Device != nil {
		fp = strings.TrimSpace(body.Device.Fingerprint)
	} else {
		fp = strings.TrimSpace(body.DeviceFingerprint)
	}
	seatID, err := h.licensingDelegate.DeactivateSeat(body.LicenseKey, fp)
	if err != nil {
		if errors.Is(err, licensing.ErrLicenseNotFound) || errors.Is(err, licensing.ErrSeatNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "errorCode": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "seatId": seatID})
}

type deviceLinkStartRequest struct {
	DeviceFingerprint string `json:"deviceFingerprint"`
	Device            *struct {
		Fingerprint string `json:"fingerprint"`
	} `json:"device"`
}

func (h *Handler) StartDeviceLink(c *gin.Context) {
	setCORS(c)
	var body deviceLinkStartRequest
	_ = c.ShouldBindJSON(&body)
	fp := strings.TrimSpace(c.GetHeader("X-RS3-Device-Fingerprint"))
	if fp == "" && body.Device != nil {
		fp = strings.TrimSpace(body.Device.Fingerprint)
	}
	if fp == "" {
		fp = strings.TrimSpace(body.DeviceFingerprint)
	}
	session, err := h.licensingDelegate.StartDeviceLink(fp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	frontendBase := strings.TrimRight(viper.GetString("oidc.frontendBaseUrl"), "/")
	if frontendBase == "" {
		frontendBase = "http://localhost:3030"
	}
	ttl := int(time.Until(session.ExpiresAt).Seconds())
	if ttl < 0 {
		ttl = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"device_code":      session.DeviceCode,
		"user_code":        session.UserCode,
		"verification_uri": frontendBase + "/device/link",
		"expiresIn":        ttl,
	})
}

type deviceLinkPollRequest struct {
	DeviceCode        string `json:"device_code"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	PublicBase        string `json:"publicBase"`
}

func (h *Handler) PollDeviceLink(c *gin.Context) {
	setCORS(c)
	var body deviceLinkPollRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	fp := strings.TrimSpace(c.GetHeader("X-RS3-Device-Fingerprint"))
	if fp == "" {
		fp = strings.TrimSpace(body.DeviceFingerprint)
	}
	result, status, err := h.licensingDelegate.PollDeviceLink(body.DeviceCode, fp, publicBaseFromRequest(c, body.PublicBase))
	if errors.Is(err, licensing.ErrDeviceLinkPending) {
		c.JSON(http.StatusOK, gin.H{"status": status})
		return
	}
	if err != nil {
		writeActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":             status,
		"signedOfflineToken": result.SignedOfflineToken,
		"addonManifestUrl":   result.AddonManifestUrl,
		"serverTime":         result.ServerTime.Format(time.RFC3339),
		"refreshAfter":       result.RefreshAfter.Format(time.RFC3339),
		"seatId":             result.SeatID,
	})
}

type confirmDeviceLinkRequest struct {
	UserCode  string `json:"userCode"`
	LicenseID string `json:"licenseId"`
}

func (h *Handler) ConfirmDeviceLink(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body confirmDeviceLinkRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	session, err := h.licensingDelegate.ConfirmDeviceLink(userID, body.UserCode, body.LicenseID)
	if err != nil {
		writeActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"userCode":   session.UserCode,
		"licenseId":  session.LicenseID,
		"status":     session.Status,
	})
}

func (h *Handler) AddonManifest(c *gin.Context) {
	setCORS(c)
	token, fp := extractAddonAuth(c)
	manifest, err := h.licensingDelegate.BuildAddonManifest(token, fp)
	if err != nil {
		writeAddonAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, manifest)
}

func (h *Handler) AddonBundle(c *gin.Context) {
	setCORS(c)
	token, fp := extractAddonAuth(c)
	enc, err := h.licensingDelegate.EncryptAddonBundle(token, fp)
	if err != nil {
		writeAddonAuthError(c, err)
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, enc)
}

type issueLicenseRequest struct {
	LmsUserID    string   `json:"lmsUserId"`
	SeatLimit    int      `json:"seatLimit"`
	Capabilities []string `json:"capabilities"`
	ExpiresAt    string   `json:"expiresAt"`
	Note         string   `json:"note"`
}

func (h *Handler) IssueLicense(c *gin.Context) {
	userID, userRole, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	allowed := []models.Role{models.SuperAdmin}
	if accessErr := h.authDelegate.UserAccess(userRole, allowed, c); accessErr != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var body issueLicenseRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.LmsUserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	input := models.IssueLicenseInput{
		LmsUserID:    strings.TrimSpace(body.LmsUserID),
		SeatLimit:    body.SeatLimit,
		Capabilities: body.Capabilities,
		Note:         body.Note,
		IssuedBy:     userID,
	}
	if body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, body.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_expires_at"})
			return
		}
		input.ExpiresAt = t
	}
	lic, err := h.licensingDelegate.IssueLicense(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, licenseToJSON(lic))
}

func (h *Handler) ListMyLicenses(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	list, err := h.licensingDelegate.ListMyLicenses(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, lic := range list {
		out = append(out, licenseToJSON(lic))
	}
	c.JSON(http.StatusOK, gin.H{"licenses": out})
}

func (h *Handler) GetLicense(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	lic, err := h.licensingDelegate.GetLicense(userID, c.Param("licenseId"))
	if err != nil {
		writeActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, licenseToJSON(lic))
}

func (h *Handler) RevokeSeat(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	err = h.licensingDelegate.RevokeSeat(userID, c.Param("licenseId"), c.Param("seatId"))
	if err != nil {
		writeActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// sessionIdentity prefers OIDC/BFF identity already placed into gin context by
// TokenAuthMiddleware (user_id / user_role). Falls back to legacy JWT Bearer
// parsing via auth.UserIdentity for AUTH_MODE=legacy_jwt.
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

func extractAddonAuth(c *gin.Context) (token, fingerprint string) {
	fingerprint = strings.TrimSpace(c.GetHeader("X-RS3-Device-Fingerprint"))
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		token = strings.TrimSpace(parts[1])
	}
	return
}

func writeActivationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, licensing.ErrDeviceFingerprint):
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_fingerprint_required", "errorCode": "DEVICE_FINGERPRINT_REQUIRED"})
	case errors.Is(err, licensing.ErrLicenseNotFound):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "INVALID_KEY", "errorCode": "INVALID_KEY"})
	case errors.Is(err, licensing.ErrSeatLimitReached):
		c.JSON(http.StatusForbidden, gin.H{"error": "SEAT_LIMIT_REACHED", "errorCode": "SEAT_LIMIT_REACHED"})
	case errors.Is(err, licensing.ErrSubscriptionExpired):
		c.JSON(http.StatusForbidden, gin.H{"error": "SUBSCRIPTION_EXPIRED", "errorCode": "SUBSCRIPTION_EXPIRED"})
	case errors.Is(err, licensing.ErrReactivationCooldown):
		c.JSON(http.StatusBadRequest, gin.H{"error": "REACTIVATION_COOLDOWN", "errorCode": "REACTIVATION_COOLDOWN"})
	case errors.Is(err, licensing.ErrDeviceLinkNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "DEVICE_LINK_NOT_FOUND", "errorCode": "DEVICE_LINK_NOT_FOUND"})
	case errors.Is(err, licensing.ErrDeviceLinkExpired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "DEVICE_LINK_EXPIRED", "errorCode": "DEVICE_LINK_EXPIRED"})
	case errors.Is(err, licensing.ErrSeatNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "SEAT_NOT_FOUND", "errorCode": "SEAT_NOT_FOUND"})
	case errors.Is(err, licensing.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
	}
}

func writeAddonAuthError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, licensing.ErrAddonAuthRequired):
		msg = "addon_auth_required"
	case errors.Is(err, licensing.ErrInvalidToken):
		msg = "invalid_token_shape"
	case errors.Is(err, licensing.ErrUnsupportedAlg):
		msg = "unsupported_alg"
	case errors.Is(err, licensing.ErrInvalidSignature):
		msg = "invalid_signature"
	case errors.Is(err, licensing.ErrLicenseExpiredToken):
		msg = "license_expired"
	case errors.Is(err, licensing.ErrDeviceBinding):
		msg = "device_binding_mismatch"
	case errors.Is(err, licensing.ErrSeatNotActive):
		msg = "seat_not_active"
	default:
		if strings.Contains(msg, "manifest_build_failed") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "manifest_build_failed"})
			return
		}
		if strings.Contains(msg, "addon_encrypt") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "addon_encrypt_failed"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"error": msg})
}

func licenseToJSON(lic *models.LicenseCore) gin.H {
	seats := make([]gin.H, 0, len(lic.Seats))
	for _, s := range lic.Seats {
		seats = append(seats, gin.H{
			"seatId":            s.SeatID,
			"deviceFingerprint": s.DeviceFingerprint,
			"label":             s.Label,
			"activatedAt":       s.ActivatedAt.Format(time.RFC3339),
			"lastSeenAt":        s.LastSeenAt.Format(time.RFC3339),
		})
	}
	return gin.H{
		"id":           lic.ID,
		"lmsUserId":    lic.LmsUserID,
		"licenseKey":   lic.LicenseKey,
		"status":       lic.Status,
		"source":       lic.Source,
		"seatLimit":    lic.SeatLimit,
		"capabilities": lic.Capabilities,
		"expiresAt":    lic.ExpiresAt.Format(time.RFC3339),
		"issuedBy":     lic.IssuedBy,
		"note":         lic.Note,
		"createdAt":    lic.CreatedAt.Format(time.RFC3339),
		"seats":        seats,
	}
}
