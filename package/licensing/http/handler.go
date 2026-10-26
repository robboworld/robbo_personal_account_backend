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
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
	"github.com/spf13/viper"
)

type Handler struct {
	authDelegate      auth.Delegate
	licensingDelegate licensing.Delegate
	projectStorage    projectPage.Gateway
}

func NewLicensingHandler(
	authDelegate auth.Delegate,
	licensingDelegate licensing.Delegate,
	projectStorage projectPage.Gateway,
) Handler {
	return Handler{
		authDelegate:      authDelegate,
		licensingDelegate: licensingDelegate,
		projectStorage:    projectStorage,
	}
}

func (h *Handler) InitLicensingRoutes(router *gin.Engine) {
	router.OPTIONS("/v1/activate", h.corsPreflight)
	router.OPTIONS("/v1/seats/deactivate", h.corsPreflight)
	router.OPTIONS("/v1/seats/release", h.corsPreflight)
	router.OPTIONS("/v1/token/refresh", h.corsPreflight)
	router.OPTIONS("/v1/device/link/start", h.corsPreflight)
	router.OPTIONS("/v1/device/link/poll", h.corsPreflight)
	router.OPTIONS("/addon/manifest.json", h.corsPreflight)
	router.OPTIONS("/addon/paid-addon.js.enc", h.corsPreflight)
	router.OPTIONS("/health/licensing", h.corsPreflight)

	router.POST("/v1/activate", h.Activate)
	router.POST("/v1/seats/deactivate", h.DeactivateSeat)
	router.POST("/v1/seats/release", h.ReleaseSeat)
	router.POST("/v1/token/refresh", h.RefreshToken)
	router.POST("/v1/device/link/start", h.StartDeviceLink)
	router.POST("/v1/device/link/poll", h.PollDeviceLink)
	router.GET("/addon/manifest.json", h.AddonManifest)
	router.GET("/addon/paid-addon.js.enc", h.AddonBundle)
	router.GET("/health/licensing", h.Health)

	api := router.Group("/licensing")
	{
		api.POST("/issue", h.IssueLicense)
		api.GET("/mine", h.ListMyLicenses)
		api.GET("/entitlements", h.GetEntitlements)
		api.GET("/interest", h.GetFeatureInterest)
		api.POST("/interest", h.SaveFeatureInterest)
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
	LicenseKey        string `json:"licenseKey"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	DeviceID          string `json:"deviceId"`
	PublicBase        string `json:"publicBase"`
	Device            *struct {
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

// deviceTokenRequest: the device's offline token and fingerprint, in the body or as
// Authorization: Bearer / X-RS3-Device-Fingerprint (the same headers as the addon download).
type deviceTokenRequest struct {
	Token             string `json:"token"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	PublicBase        string `json:"publicBase"`
}

func extractDeviceToken(c *gin.Context, body deviceTokenRequest) (token, fingerprint string) {
	token, fingerprint = extractAddonAuth(c)
	if token == "" {
		token = strings.TrimSpace(body.Token)
	}
	if fingerprint == "" {
		fingerprint = strings.TrimSpace(body.DeviceFingerprint)
	}
	return
}

// RefreshToken: POST /v1/token/refresh — a new offline token with the licence's current terms, or
// the reason it no longer applies to this device.
func (h *Handler) RefreshToken(c *gin.Context) {
	setCORS(c)
	var body deviceTokenRequest
	_ = c.ShouldBindJSON(&body)
	token, fp := extractDeviceToken(c, body)
	result, err := h.licensingDelegate.RefreshToken(token, fp, publicBaseFromRequest(c, body.PublicBase))
	if err != nil {
		writeDeviceTokenError(c, err)
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

// ReleaseSeat: POST /v1/seats/release — frees this device's seat by its offline token.
func (h *Handler) ReleaseSeat(c *gin.Context) {
	setCORS(c)
	var body deviceTokenRequest
	_ = c.ShouldBindJSON(&body)
	token, fp := extractDeviceToken(c, body)
	seatID, err := h.licensingDelegate.ReleaseSeat(token, fp)
	if err != nil {
		writeDeviceTokenError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "seatId": seatID})
}

// writeDeviceTokenError answers token refresh / seat release with a "reason" the editor shows:
// expired | revoked | seat_removed | other_device | invalid.
func writeDeviceTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, licensing.ErrSubscriptionExpired):
		c.JSON(http.StatusForbidden, gin.H{"error": "SUBSCRIPTION_EXPIRED", "reason": "expired"})
	case errors.Is(err, licensing.ErrLicenseRevoked):
		c.JSON(http.StatusForbidden, gin.H{"error": "license_revoked", "reason": "revoked"})
	case errors.Is(err, licensing.ErrSeatRemoved):
		c.JSON(http.StatusForbidden, gin.H{"error": "seat_removed", "reason": "seat_removed"})
	case errors.Is(err, licensing.ErrSeatNotFound):
		// Release: the seat is already free.
		c.JSON(http.StatusNotFound, gin.H{"error": "SEAT_NOT_FOUND", "errorCode": "SEAT_NOT_FOUND", "reason": "seat_removed"})
	case errors.Is(err, licensing.ErrDeviceBinding):
		c.JSON(http.StatusForbidden, gin.H{"error": "device_binding_mismatch", "reason": "other_device"})
	case errors.Is(err, licensing.ErrDeviceFingerprint):
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_fingerprint_required", "errorCode": "DEVICE_FINGERPRINT_REQUIRED"})
	case errors.Is(err, licensing.ErrAddonAuthRequired),
		errors.Is(err, licensing.ErrInvalidToken),
		errors.Is(err, licensing.ErrUnsupportedAlg),
		errors.Is(err, licensing.ErrInvalidSignature):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "reason": "invalid"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
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
		"ok":        true,
		"userCode":  session.UserCode,
		"licenseId": session.LicenseID,
		"status":    session.Status,
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
	LmsUserID      string   `json:"lmsUserId"`
	LmsUsername    string   `json:"lmsUsername"`
	SeatLimit      int      `json:"seatLimit"`
	Capabilities   []string `json:"capabilities"`
	CloudQuotaMB   int      `json:"cloudQuotaMb"`
	StorageQuotaMB int      `json:"storageQuotaMb"`
	SessionLimit   int      `json:"sessionLimit"`
	ExpiresAt      string   `json:"expiresAt"`
	Note           string   `json:"note"`
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
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	lmsUserID := strings.TrimSpace(body.LmsUserID)
	if lmsUserID == "" {
		username := strings.TrimSpace(body.LmsUsername)
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
			return
		}
		resolved, resolveErr := usersearch.ResolveUsernameToID(username)
		if resolveErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": resolveErr.Error()})
			return
		}
		lmsUserID = resolved
	}
	input := models.IssueLicenseInput{
		LmsUserID:      lmsUserID,
		SeatLimit:      body.SeatLimit,
		Capabilities:   body.Capabilities,
		CloudQuotaMB:   body.CloudQuotaMB,
		StorageQuotaMB: body.StorageQuotaMB,
		SessionLimit:   body.SessionLimit,
		Note:           body.Note,
		IssuedBy:       userID,
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

func (h *Handler) GetEntitlements(c *gin.Context) {
	userID, role, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ent, err := h.licensingDelegate.ResolveEntitlements(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	licensing.ApplyAdminSessionSeatExemption(&ent, role)
	var usedBytes int64
	var projectCount int64
	if h.projectStorage != nil {
		usedBytes, err = h.projectStorage.GetTotalStorageBytesForOwner(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		projectCount, err = h.projectStorage.CountProjectsByOwner(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	tariffName := ent.TariffName
	if tariffName == "" {
		tariffName = "Free"
	}
	var expiresAt interface{}
	if !ent.ExpiresAt.IsZero() {
		expiresAt = ent.ExpiresAt.UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, gin.H{
		"expiresAt":        expiresAt,
		"tariffName":       tariffName,
		"hasLicense":       ent.HasLicense,
		"cloudQuotaMb":     ent.MaxProjectSizeMB,
		"maxProjectSizeMb": ent.MaxProjectSizeMB,
		"storageQuotaMb":   ent.StorageQuotaMB,
		"projectCount":     projectCount,
		"usedBytes":        usedBytes,
		"sessionLimit":     ent.SessionLimit,
		"seatLimit":        ent.SeatLimit,
		"capabilities":     ent.Capabilities,
	})
}

type featureInterestRequest struct {
	Role    string `json:"role"`
	Feature string `json:"feature"`
	Other   string `json:"other"`
	Notify  bool   `json:"notify"`
	Source  string `json:"source"`
}

// GetFeatureInterest: GET /licensing/interest — the account's answer to the paid-edition
// survey ({"answer": null} before the first answer).
func (h *Handler) GetFeatureInterest(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	answer, err := h.licensingDelegate.GetFeatureInterest(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if answer == nil {
		c.JSON(http.StatusOK, gin.H{"answer": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"answer": gin.H{
		"role":      answer.Role,
		"feature":   answer.Feature,
		"other":     answer.Other,
		"notify":    answer.Notify,
		"updatedAt": answer.UpdatedAt.UTC().Format(time.RFC3339),
	}})
}

// SaveFeatureInterest: POST /licensing/interest — stores the account's answer (replaces an
// earlier one).
func (h *Handler) SaveFeatureInterest(c *gin.Context) {
	userID, _, err := h.sessionIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body featureInterestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	err = h.licensingDelegate.SaveFeatureInterest(models.FeatureInterestCore{
		LmsUserID: userID,
		Role:      body.Role,
		Feature:   body.Feature,
		Other:     body.Other,
		Notify:    body.Notify,
		Source:    body.Source,
	})
	if errors.Is(err, licensing.ErrBadInterest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
		"id":             lic.ID,
		"lmsUserId":      lic.LmsUserID,
		"licenseKey":     lic.LicenseKey,
		"status":         lic.Status,
		"source":         lic.Source,
		"seatLimit":      lic.SeatLimit,
		"capabilities":   lic.Capabilities,
		"cloudQuotaMb":   lic.CloudQuotaMB,
		"storageQuotaMb": lic.StorageQuotaMB,
		"sessionLimit":   lic.SessionLimit,
		"expiresAt":      lic.ExpiresAt.Format(time.RFC3339),
		"issuedBy":       lic.IssuedBy,
		"note":           lic.Note,
		"createdAt":      lic.CreatedAt.Format(time.RFC3339),
		"seats":          seats,
	}
}
