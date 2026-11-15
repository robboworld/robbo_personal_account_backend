package usecase

import (
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

func (u *LicensingUseCaseImpl) SaveFeatureInterest(answer models.FeatureInterestCore) error {
	answer, err := licensing.NormalizeInterest(answer)
	if err != nil {
		return err
	}
	return u.gateway.SaveFeatureInterest(&answer)
}

// GetFeatureInterest returns nil, nil when the account has not answered yet.
func (u *LicensingUseCaseImpl) GetFeatureInterest(lmsUserID string) (*models.FeatureInterestCore, error) {
	answer, err := u.gateway.GetFeatureInterest(lmsUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return answer, err
}

// ListFeatureInterest returns a page of answers for the SuperAdmin view; limit is clamped to 1..200.
func (u *LicensingUseCaseImpl) ListFeatureInterest(limit, offset int) ([]models.FeatureInterestCore, int64, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return u.gateway.ListFeatureInterest(limit, offset)
}
