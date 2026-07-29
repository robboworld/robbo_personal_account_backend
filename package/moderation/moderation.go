package moderation

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// Gateway persists ban audit rows in Licensing Postgres.
type Gateway interface {
	CreateBan(ban *models.UserBanCore) (*models.UserBanCore, error)
	GetActiveBan(lmsUserID string) (*models.UserBanCore, error)
	GetOpenBan(lmsUserID string) (*models.UserBanCore, error)
	ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error)
	CloseBan(banID, unbannedBy, unbanReason string) (*models.UserBanCore, error)
	ListExpiredActiveBans(now time.Time, limit int) ([]*models.UserBanCore, error)
}

// UseCase orchestrates LMS is_active toggles, ban audit, and session revoke.
type UseCase interface {
	BanUser(callerID, targetID, reason string, expiresAt *time.Time) (*models.UserBanCore, error)
	UnbanUser(callerID, targetID, reason string) (*models.UserBanCore, error)
	GetActiveBan(lmsUserID string) (*models.UserBanCore, error)
	ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error)
	ProcessExpiredBans() (int, error)
}

// Delegate is the GraphQL-facing layer.
type Delegate interface {
	BanUser(callerID, targetID, reason string, expiresAt *time.Time) (*models.UserBanCore, error)
	UnbanUser(callerID, targetID, reason string) (*models.UserBanCore, error)
	GetActiveBan(lmsUserID string) (*models.UserBanCore, error)
	ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error)
}
