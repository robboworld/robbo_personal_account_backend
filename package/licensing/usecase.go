package licensing

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

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
}
