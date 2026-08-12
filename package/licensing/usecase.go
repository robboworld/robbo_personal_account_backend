package licensing

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// UseCase contains business logic for licensing and activation.
type UseCase interface {
	IssueLicense(input models.IssueLicenseInput) (*models.LicenseCore, error)
	ListMyLicenses(lmsUserID string) ([]*models.LicenseCore, error)
	GetLicense(lmsUserID, licenseID string) (*models.LicenseCore, error)
	RevokeSeat(lmsUserID, licenseID, seatID string) error

	Activate(licenseKey, fingerprint, publicBase string) (*models.ActivateResult, error)
	DeactivateSeat(licenseKey, fingerprint string) (seatID string, err error)

	StartDeviceLink(fingerprint string) (*models.DeviceLinkSessionCore, error)
	ConfirmDeviceLink(lmsUserID, userCode, licenseID string) (*models.DeviceLinkSessionCore, error)
	PollDeviceLink(deviceCode, fingerprint, publicBase string) (*models.ActivateResult, string, error)

	BuildAddonManifest(token, fingerprint string) (map[string]interface{}, error)
	EncryptAddonBundle(token, fingerprint string) (string, error)

	// Concurrent web-login sessions (tariff session_limit).
	BeginLoginSession(lmsUserID, authMode, userAgent, ipAddress string, ttl time.Duration, role models.Role) (*models.UserSessionCore, error)
	TouchLoginSession(sessionKey string) error
	RevokeLoginSession(sessionKey string) error
	ListLoginSessions(lmsUserID string) ([]*models.UserSessionCore, error)
	RevokeLoginSessionByID(lmsUserID, sessionID string) error
	ResolveEntitlements(lmsUserID string) (Entitlements, error)
}
