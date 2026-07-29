package delegate

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"go.uber.org/fx"
)

// ModerationDelegateImpl is a thin pass-through to the ban usecase.
type ModerationDelegateImpl struct {
	UseCase moderation.UseCase
}

// ModerationDelegateModule is the FX output.
type ModerationDelegateModule struct {
	fx.Out
	moderation.Delegate
}

// SetupModerationDelegate wires the GraphQL-facing delegate.
func SetupModerationDelegate(usecase moderation.UseCase) ModerationDelegateModule {
	return ModerationDelegateModule{
		Delegate: &ModerationDelegateImpl{UseCase: usecase},
	}
}

func (d *ModerationDelegateImpl) BanUser(callerID, targetID, reason string, expiresAt *time.Time) (*models.UserBanCore, error) {
	return d.UseCase.BanUser(callerID, targetID, reason, expiresAt)
}

func (d *ModerationDelegateImpl) UnbanUser(callerID, targetID, reason string) (*models.UserBanCore, error) {
	return d.UseCase.UnbanUser(callerID, targetID, reason)
}

func (d *ModerationDelegateImpl) GetActiveBan(lmsUserID string) (*models.UserBanCore, error) {
	return d.UseCase.GetActiveBan(lmsUserID)
}

func (d *ModerationDelegateImpl) ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error) {
	return d.UseCase.ListBanHistory(lmsUserID, limit)
}
