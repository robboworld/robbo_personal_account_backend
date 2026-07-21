package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type LicensingGatewayImpl struct {
	db *gorm.DB
}

type LicensingGatewayModule struct {
	fx.Out
	licensing.Gateway
}

func SetupLicensingGateway(postgresClient db_client.PostgresClient) LicensingGatewayModule {
	_ = postgresClient
	dsn := viper.GetString("licensingPostgres.postgresDsn")
	if dsn == "" {
		panic("licensingPostgres.postgresDsn (or env LICENSING_POSTGRES_DSN) is required")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return LicensingGatewayModule{
		Gateway: &LicensingGatewayImpl{db: db},
	}
}

func (g *LicensingGatewayImpl) CreateLicense(license *models.LicenseCore) (*models.LicenseCore, error) {
	dbRow := models.LicenseDB{
		LmsUserID:    license.LmsUserID,
		LicenseKey:   license.LicenseKey,
		Status:       license.Status,
		Source:       license.Source,
		SeatLimit:    license.SeatLimit,
		Capabilities: pq.StringArray(license.Capabilities),
		ExpiresAt:    license.ExpiresAt,
		Note:         license.Note,
	}
	if license.IssuedBy != "" {
		dbRow.IssuedBy = &license.IssuedBy
	}
	if license.ProductID != "" {
		dbRow.ProductID = &license.ProductID
	}
	if err := g.db.Create(&dbRow).Error; err != nil {
		return nil, err
	}
	return dbRow.ToCore(), nil
}

func (g *LicensingGatewayImpl) GetLicenseByKey(licenseKey string) (*models.LicenseCore, error) {
	var dbRow models.LicenseDB
	err := g.db.Where("license_key = ?", licenseKey).First(&dbRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, licensing.ErrLicenseNotFound
	}
	if err != nil {
		return nil, err
	}
	core := dbRow.ToCore()
	seats, err := g.ListSeats(core.ID)
	if err != nil {
		return nil, err
	}
	for _, s := range seats {
		core.Seats = append(core.Seats, *s)
	}
	return core, nil
}

func (g *LicensingGatewayImpl) GetLicenseByID(id string) (*models.LicenseCore, error) {
	var dbRow models.LicenseDB
	err := g.db.Where("id = ?", id).First(&dbRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, licensing.ErrLicenseNotFound
	}
	if err != nil {
		return nil, err
	}
	core := dbRow.ToCore()
	seats, err := g.ListSeats(core.ID)
	if err != nil {
		return nil, err
	}
	for _, s := range seats {
		core.Seats = append(core.Seats, *s)
	}
	return core, nil
}

func (g *LicensingGatewayImpl) ListLicensesByUser(lmsUserID string) ([]*models.LicenseCore, error) {
	var rows []models.LicenseDB
	if err := g.db.Where("lms_user_id = ?", lmsUserID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*models.LicenseCore, 0, len(rows))
	for i := range rows {
		core := rows[i].ToCore()
		seats, err := g.ListSeats(core.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range seats {
			core.Seats = append(core.Seats, *s)
		}
		out = append(out, core)
	}
	return out, nil
}

func (g *LicensingGatewayImpl) UpdateLicense(license *models.LicenseCore) error {
	updates := map[string]interface{}{
		"status":     license.Status,
		"seat_limit": license.SeatLimit,
		"note":       license.Note,
		"expires_at": license.ExpiresAt,
		"updated_at": time.Now().UTC(),
	}
	if license.LastSeatChangeAt != nil {
		updates["last_seat_change_at"] = *license.LastSeatChangeAt
	}
	return g.db.Model(&models.LicenseDB{}).Where("id = ?", license.ID).Updates(updates).Error
}

func (g *LicensingGatewayImpl) ListSeats(licenseID string) ([]*models.SeatCore, error) {
	var rows []models.SeatDB
	if err := g.db.Where("license_id = ?", licenseID).Order("activated_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*models.SeatCore, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToCore())
	}
	return out, nil
}

func (g *LicensingGatewayImpl) GetSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error) {
	var row models.SeatDB
	err := g.db.Where("license_id = ? AND device_fingerprint = ?", licenseID, fingerprint).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, licensing.ErrSeatNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *LicensingGatewayImpl) CreateSeat(seat *models.SeatCore) (*models.SeatCore, error) {
	if seat.SeatID == "" {
		seat.SeatID = newSeatID()
	}
	now := time.Now().UTC()
	if seat.ActivatedAt.IsZero() {
		seat.ActivatedAt = now
	}
	if seat.LastSeenAt.IsZero() {
		seat.LastSeenAt = now
	}
	row := models.SeatDB{
		LicenseID:         seat.LicenseID,
		SeatID:            seat.SeatID,
		DeviceFingerprint: seat.DeviceFingerprint,
		Label:             seat.Label,
		ActivatedAt:       seat.ActivatedAt,
		LastSeenAt:        seat.LastSeenAt,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *LicensingGatewayImpl) TouchSeat(seatID string, lastSeenAt time.Time) error {
	return g.db.Model(&models.SeatDB{}).Where("seat_id = ?", seatID).Update("last_seen_at", lastSeenAt).Error
}

func (g *LicensingGatewayImpl) DeleteSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error) {
	seat, err := g.GetSeatByFingerprint(licenseID, fingerprint)
	if err != nil {
		return nil, err
	}
	if err := g.db.Where("license_id = ? AND device_fingerprint = ?", licenseID, fingerprint).Delete(&models.SeatDB{}).Error; err != nil {
		return nil, err
	}
	return seat, nil
}

func (g *LicensingGatewayImpl) CountSeats(licenseID string) (int64, error) {
	var count int64
	err := g.db.Model(&models.SeatDB{}).Where("license_id = ?", licenseID).Count(&count).Error
	return count, err
}

func (g *LicensingGatewayImpl) CreateDeviceLink(session *models.DeviceLinkSessionCore) (*models.DeviceLinkSessionCore, error) {
	row := models.DeviceLinkSessionDB{
		DeviceCode:        session.DeviceCode,
		UserCode:          session.UserCode,
		Status:            session.Status,
		DeviceFingerprint: session.DeviceFingerprint,
		ExpiresAt:         session.ExpiresAt,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *LicensingGatewayImpl) GetDeviceLinkByDeviceCode(deviceCode string) (*models.DeviceLinkSessionCore, error) {
	var row models.DeviceLinkSessionDB
	err := g.db.Where("device_code = ?", deviceCode).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, licensing.ErrDeviceLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *LicensingGatewayImpl) GetDeviceLinkByUserCode(userCode string) (*models.DeviceLinkSessionCore, error) {
	var row models.DeviceLinkSessionDB
	err := g.db.Where("user_code = ?", strings.ToUpper(userCode)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, licensing.ErrDeviceLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *LicensingGatewayImpl) UpdateDeviceLink(session *models.DeviceLinkSessionCore) error {
	updates := map[string]interface{}{
		"status":             session.Status,
		"device_fingerprint": session.DeviceFingerprint,
	}
	if session.LmsUserID != "" {
		updates["lms_user_id"] = session.LmsUserID
	}
	if session.LicenseID != "" {
		updates["license_id"] = session.LicenseID
	}
	if session.SeatID != "" {
		updates["seat_id"] = session.SeatID
	}
	if session.ConfirmedAt != nil {
		updates["confirmed_at"] = *session.ConfirmedAt
	}
	return g.db.Model(&models.DeviceLinkSessionDB{}).Where("device_code = ?", session.DeviceCode).Updates(updates).Error
}

func newSeatID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "seat-" + hex.EncodeToString(b)
}

// NewLicenseKey generates a human-readable license key RS3-XXXX-XXXX-XXXX.
func NewLicenseKey() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hexStr := strings.ToUpper(hex.EncodeToString(b))
	return fmt.Sprintf("RS3-%s-%s-%s", hexStr[0:4], hexStr[4:8], hexStr[8:12]), nil
}
