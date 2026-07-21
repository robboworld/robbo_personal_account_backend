package delegate

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"go.uber.org/fx"
)

type LicensingDelegateImpl struct {
	usecase licensing.UseCase
}

type LicensingDelegateModule struct {
	fx.Out
	licensing.Delegate
}

func SetupLicensingDelegate(usecase licensing.UseCase) LicensingDelegateModule {
	return LicensingDelegateModule{
		Delegate: &LicensingDelegateImpl{usecase: usecase},
	}
}

func (d *LicensingDelegateImpl) IssueLicense(input models.IssueLicenseInput) (*models.LicenseCore, error) {
	return d.usecase.IssueLicense(input)
}

func (d *LicensingDelegateImpl) ListMyLicenses(lmsUserID string) ([]*models.LicenseCore, error) {
	return d.usecase.ListMyLicenses(lmsUserID)
}

func (d *LicensingDelegateImpl) GetLicense(lmsUserID, licenseID string) (*models.LicenseCore, error) {
	return d.usecase.GetLicense(lmsUserID, licenseID)
}

func (d *LicensingDelegateImpl) RevokeSeat(lmsUserID, licenseID, seatID string) error {
	return d.usecase.RevokeSeat(lmsUserID, licenseID, seatID)
}

func (d *LicensingDelegateImpl) Activate(licenseKey, fingerprint, publicBase string) (*models.ActivateResult, error) {
	return d.usecase.Activate(licenseKey, fingerprint, publicBase)
}

func (d *LicensingDelegateImpl) DeactivateSeat(licenseKey, fingerprint string) (string, error) {
	return d.usecase.DeactivateSeat(licenseKey, fingerprint)
}

func (d *LicensingDelegateImpl) StartDeviceLink(fingerprint string) (*models.DeviceLinkSessionCore, error) {
	return d.usecase.StartDeviceLink(fingerprint)
}

func (d *LicensingDelegateImpl) ConfirmDeviceLink(lmsUserID, userCode, licenseID string) (*models.DeviceLinkSessionCore, error) {
	return d.usecase.ConfirmDeviceLink(lmsUserID, userCode, licenseID)
}

func (d *LicensingDelegateImpl) PollDeviceLink(deviceCode, fingerprint, publicBase string) (*models.ActivateResult, string, error) {
	return d.usecase.PollDeviceLink(deviceCode, fingerprint, publicBase)
}

func (d *LicensingDelegateImpl) BuildAddonManifest(token, fingerprint string) (map[string]interface{}, error) {
	return d.usecase.BuildAddonManifest(token, fingerprint)
}

func (d *LicensingDelegateImpl) EncryptAddonBundle(token, fingerprint string) (string, error) {
	return d.usecase.EncryptAddonBundle(token, fingerprint)
}
