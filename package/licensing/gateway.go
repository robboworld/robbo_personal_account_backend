package licensing

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// Gateway persists licenses, seats and device-link sessions.
type Gateway interface {
	CreateLicense(license *models.LicenseCore) (*models.LicenseCore, error)
	GetLicenseByKey(licenseKey string) (*models.LicenseCore, error)
	GetLicenseByID(id string) (*models.LicenseCore, error)
	ListLicensesByUser(lmsUserID string) ([]*models.LicenseCore, error)
	UpdateLicense(license *models.LicenseCore) error

	ListSeats(licenseID string) ([]*models.SeatCore, error)
	GetSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error)
	CreateSeat(seat *models.SeatCore) (*models.SeatCore, error)
	TouchSeat(seatID string, lastSeenAt time.Time) error
	DeleteSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error)
	CountSeats(licenseID string) (int64, error)

	CreateDeviceLink(session *models.DeviceLinkSessionCore) (*models.DeviceLinkSessionCore, error)
	GetDeviceLinkByDeviceCode(deviceCode string) (*models.DeviceLinkSessionCore, error)
	GetDeviceLinkByUserCode(userCode string) (*models.DeviceLinkSessionCore, error)
	UpdateDeviceLink(session *models.DeviceLinkSessionCore) error
}
