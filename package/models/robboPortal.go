package models

import "time"

type RobboPortalUserLinkDB struct {
	ID             int64      `gorm:"primaryKey"`
	EdxUserID      *string    `gorm:"column:edx_user_id"`
	LegacyLKUserID *string    `gorm:"column:legacy_lk_user_id"`
	OIDCSub        *string    `gorm:"column:oidc_sub"`
	Email          string     `gorm:"column:email;not null"`
	DisplayName    *string    `gorm:"column:display_name"`
	LastLoginAt    *time.Time `gorm:"column:last_login_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (RobboPortalUserLinkDB) TableName() string { return "robbo_portal_user_link" }

type RobboPortalRoleDB struct {
	ID         int64     `gorm:"primaryKey"`
	UserLinkID int64     `gorm:"column:user_link_id;not null"`
	RoleCode   string    `gorm:"column:role_code;not null"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (RobboPortalRoleDB) TableName() string { return "robbo_portal_role" }

type RobboPortalIntegrationOutboxDB struct {
	ID             int64      `gorm:"primaryKey"`
	IdempotencyKey string     `gorm:"column:idempotency_key;uniqueIndex;not null"`
	EventType      string     `gorm:"column:event_type;not null"`
	Payload        string     `gorm:"column:payload;type:jsonb;not null"`
	Status         string     `gorm:"column:status;not null;default:pending"`
	Attempts       int        `gorm:"column:attempts;not null;default:0"`
	LastError      *string    `gorm:"column:last_error"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ProcessedAt    *time.Time `gorm:"column:processed_at"`
}

func (RobboPortalIntegrationOutboxDB) TableName() string { return "robbo_portal_integration_outbox" }

type OidcSessionClaims struct {
	Sub       string `json:"sub"`
	EdxUserID string `json:"edx_user_id"`
	Email     string `json:"email"`
	Role      uint   `json:"role"`
	// Sid is the concurrent-session tracking key (lk_user_sessions.session_key).
	Sid string `json:"sid,omitempty"`
}
