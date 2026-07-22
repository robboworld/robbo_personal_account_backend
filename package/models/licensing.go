package models

import (
	"time"

	"github.com/lib/pq"
)

const (
	LicenseStatusActive   = "active"
	LicenseStatusRevoked  = "revoked"
	LicenseStatusExpired  = "expired"
	LicenseSourceAdmin    = "admin_grant"
	LicenseSourceOrder    = "order"
	LicenseSourceBonus    = "course_bonus"
	DeviceLinkPending     = "pending"
	DeviceLinkConfirmed   = "confirmed"
	DeviceLinkExpired     = "expired"
	DeviceLinkConsumed    = "consumed"
	CapabilityPremiumAuto = "premium.auto_update"
)

// LicenseDB is the GORM model for licenses table.
type LicenseDB struct {
	ID               string         `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	LmsUserID        string         `gorm:"column:lms_user_id;not null;index"`
	LicenseKey       string         `gorm:"column:license_key;uniqueIndex;not null"`
	Status           string         `gorm:"column:status;not null;default:active"`
	Source           string         `gorm:"column:source;not null;default:admin_grant"`
	SeatLimit        int            `gorm:"column:seat_limit;not null;default:1"`
	Capabilities     pq.StringArray `gorm:"column:capabilities;type:text[];not null"`
	ExpiresAt        time.Time      `gorm:"column:expires_at;not null"`
	ProductID        *string        `gorm:"column:product_id;type:uuid"`
	IssuedBy         *string        `gorm:"column:issued_by"`
	Note             string         `gorm:"column:note;not null;default:''"`
	LastSeatChangeAt *time.Time     `gorm:"column:last_seat_change_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (LicenseDB) TableName() string { return "licenses" }

// SeatDB is the GORM model for seats table.
type SeatDB struct {
	ID                string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	LicenseID         string    `gorm:"column:license_id;type:uuid;not null;index"`
	SeatID            string    `gorm:"column:seat_id;uniqueIndex;not null"`
	DeviceFingerprint string    `gorm:"column:device_fingerprint;not null"`
	Label             string    `gorm:"column:label;not null;default:''"`
	ActivatedAt       time.Time `gorm:"column:activated_at;not null"`
	LastSeenAt        time.Time `gorm:"column:last_seen_at;not null"`
}

func (SeatDB) TableName() string { return "seats" }

// DeviceLinkSessionDB is a temporary device authorization session.
type DeviceLinkSessionDB struct {
	ID                string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DeviceCode        string     `gorm:"column:device_code;uniqueIndex;not null"`
	UserCode          string     `gorm:"column:user_code;uniqueIndex;not null"`
	Status            string     `gorm:"column:status;not null;default:pending"`
	DeviceFingerprint string     `gorm:"column:device_fingerprint;not null;default:''"`
	LmsUserID         *string    `gorm:"column:lms_user_id"`
	LicenseID         *string    `gorm:"column:license_id;type:uuid"`
	SeatID            *string    `gorm:"column:seat_id"`
	ExpiresAt         time.Time  `gorm:"column:expires_at;not null"`
	ConfirmedAt       *time.Time `gorm:"column:confirmed_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (DeviceLinkSessionDB) TableName() string { return "device_link_sessions" }

// LicenseCore is the domain model used across layers.
type LicenseCore struct {
	ID               string
	LmsUserID        string
	LicenseKey       string
	Status           string
	Source           string
	SeatLimit        int
	Capabilities     []string
	ExpiresAt        time.Time
	ProductID        string
	IssuedBy         string
	Note             string
	LastSeatChangeAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Seats            []SeatCore
}

type SeatCore struct {
	ID                string
	LicenseID         string
	SeatID            string
	DeviceFingerprint string
	Label             string
	ActivatedAt       time.Time
	LastSeenAt        time.Time
}

type DeviceLinkSessionCore struct {
	ID                string
	DeviceCode        string
	UserCode          string
	Status            string
	DeviceFingerprint string
	LmsUserID         string
	LicenseID         string
	SeatID            string
	ExpiresAt         time.Time
	ConfirmedAt       *time.Time
	CreatedAt         time.Time
}

type IssueLicenseInput struct {
	LmsUserID    string
	SeatLimit    int
	Capabilities []string
	ExpiresAt    time.Time
	Note         string
	IssuedBy     string
	ProductID    string
	Source       string
}

type ActivateResult struct {
	SignedOfflineToken string
	AddonManifestUrl   string
	ServerTime         time.Time
	RefreshAfter       time.Time
	SeatID             string
	LicenseID          string
	EntitlementID      string
}

func (db *LicenseDB) ToCore() *LicenseCore {
	caps := make([]string, len(db.Capabilities))
	copy(caps, db.Capabilities)
	core := &LicenseCore{
		ID:               db.ID,
		LmsUserID:        db.LmsUserID,
		LicenseKey:       db.LicenseKey,
		Status:           db.Status,
		Source:           db.Source,
		SeatLimit:        db.SeatLimit,
		Capabilities:     caps,
		ExpiresAt:        db.ExpiresAt,
		Note:             db.Note,
		LastSeatChangeAt: db.LastSeatChangeAt,
		CreatedAt:        db.CreatedAt,
		UpdatedAt:        db.UpdatedAt,
	}
	if db.ProductID != nil {
		core.ProductID = *db.ProductID
	}
	if db.IssuedBy != nil {
		core.IssuedBy = *db.IssuedBy
	}
	return core
}

func (db *SeatDB) ToCore() *SeatCore {
	return &SeatCore{
		ID:                db.ID,
		LicenseID:         db.LicenseID,
		SeatID:            db.SeatID,
		DeviceFingerprint: db.DeviceFingerprint,
		Label:             db.Label,
		ActivatedAt:       db.ActivatedAt,
		LastSeenAt:        db.LastSeenAt,
	}
}

func (db *DeviceLinkSessionDB) ToCore() *DeviceLinkSessionCore {
	core := &DeviceLinkSessionCore{
		ID:                db.ID,
		DeviceCode:        db.DeviceCode,
		UserCode:          db.UserCode,
		Status:            db.Status,
		DeviceFingerprint: db.DeviceFingerprint,
		ExpiresAt:         db.ExpiresAt,
		ConfirmedAt:       db.ConfirmedAt,
		CreatedAt:         db.CreatedAt,
	}
	if db.LmsUserID != nil {
		core.LmsUserID = *db.LmsUserID
	}
	if db.LicenseID != nil {
		core.LicenseID = *db.LicenseID
	}
	if db.SeatID != nil {
		core.SeatID = *db.SeatID
	}
	return core
}
