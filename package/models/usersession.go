package models

import "time"

// UserSessionDB is the GORM model for lk_user_sessions table (Licensing PG).
// Tracks web login sessions (LK site + RS3 Web, same backend/cookies) so
// paid tariffs can enforce a concurrent-session limit.
type UserSessionDB struct {
	ID         string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	LmsUserID  string     `gorm:"column:lms_user_id;not null;index"`
	SessionKey string     `gorm:"column:session_key;uniqueIndex;not null"`
	AuthMode   string     `gorm:"column:auth_mode;not null;default:''"`
	UserAgent  string     `gorm:"column:user_agent;not null;default:''"`
	IPAddress  string     `gorm:"column:ip_address;not null;default:''"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
	LastSeenAt time.Time  `gorm:"column:last_seen_at;not null"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
}

func (UserSessionDB) TableName() string { return "lk_user_sessions" }

// UserSessionCore is the domain model used across layers.
type UserSessionCore struct {
	ID         string
	LmsUserID  string
	SessionKey string
	AuthMode   string
	UserAgent  string
	IPAddress  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

func (db *UserSessionDB) ToCore() *UserSessionCore {
	return &UserSessionCore{
		ID:         db.ID,
		LmsUserID:  db.LmsUserID,
		SessionKey: db.SessionKey,
		AuthMode:   db.AuthMode,
		UserAgent:  db.UserAgent,
		IPAddress:  db.IPAddress,
		CreatedAt:  db.CreatedAt,
		LastSeenAt: db.LastSeenAt,
		ExpiresAt:  db.ExpiresAt,
		RevokedAt:  db.RevokedAt,
	}
}
