package models

import "time"

// UserBanDB is the GORM model for lk_user_bans (Licensing PG).
type UserBanDB struct {
	ID                   string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	LmsUserID            string     `gorm:"column:lms_user_id;not null;index"`
	Reason               string     `gorm:"column:reason;not null"`
	BannedByLmsUserID    string     `gorm:"column:banned_by_lms_user_id;not null"`
	BannedAt             time.Time  `gorm:"column:banned_at;not null"`
	ExpiresAt            *time.Time `gorm:"column:expires_at"`
	UnbannedAt           *time.Time `gorm:"column:unbanned_at"`
	UnbannedByLmsUserID  *string    `gorm:"column:unbanned_by_lms_user_id"`
	UnbanReason          *string    `gorm:"column:unban_reason"`
	RevokedSessions      int        `gorm:"column:revoked_sessions;not null;default:0"`
}

func (UserBanDB) TableName() string { return "lk_user_bans" }

// UserBanCore is the domain model used across layers.
type UserBanCore struct {
	ID                  string
	LmsUserID           string
	Reason              string
	BannedByLmsUserID   string
	BannedAt            time.Time
	ExpiresAt           *time.Time
	UnbannedAt          *time.Time
	UnbannedByLmsUserID *string
	UnbanReason         *string
	RevokedSessions     int
}

// IsActive reports whether this ban record currently blocks the user.
func (b *UserBanCore) IsActive(now time.Time) bool {
	if b == nil || b.UnbannedAt != nil {
		return false
	}
	if b.ExpiresAt != nil && !b.ExpiresAt.After(now) {
		return false
	}
	return true
}

// IsPermanent is true when the ban has no expiry.
func (b *UserBanCore) IsPermanent() bool {
	return b != nil && b.ExpiresAt == nil
}

func (db *UserBanDB) ToCore() *UserBanCore {
	return &UserBanCore{
		ID:                  db.ID,
		LmsUserID:           db.LmsUserID,
		Reason:              db.Reason,
		BannedByLmsUserID:   db.BannedByLmsUserID,
		BannedAt:            db.BannedAt,
		ExpiresAt:           db.ExpiresAt,
		UnbannedAt:          db.UnbannedAt,
		UnbannedByLmsUserID: db.UnbannedByLmsUserID,
		UnbanReason:         db.UnbanReason,
		RevokedSessions:     db.RevokedSessions,
	}
}
