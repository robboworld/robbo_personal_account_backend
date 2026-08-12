package streak

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

// Gateway persists login streak in LMS auth_userprofile.meta JSON.
type Gateway interface {
	GetByUserID(lmsUserID string) (*models.UserLoginStreakDB, error)
	Upsert(row *models.UserLoginStreakDB) error
}

// UseCase applies daily-visit streak rules.
type UseCase interface {
	// RecordVisit updates streak for a calendar day in tz (IANA). Idempotent per day.
	RecordVisit(lmsUserID, timezone string) (*models.LoginStreakHTTP, error)
	// Get returns current streak (zeros if missing).
	Get(lmsUserID string) (*models.LoginStreakHTTP, error)
	// Increment bumps current streak by 1 (dev/test helper).
	Increment(lmsUserID string) (*models.LoginStreakHTTP, error)
}
