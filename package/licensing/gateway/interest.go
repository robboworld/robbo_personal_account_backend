package gateway

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm/clause"
)

// SaveFeatureInterest stores the account's answer; a repeat answer replaces the previous one.
func (g *LicensingGatewayImpl) SaveFeatureInterest(answer *models.FeatureInterestCore) error {
	row := models.FeatureInterestDB{
		LmsUserID: answer.LmsUserID,
		Role:      answer.Role,
		Feature:   answer.Feature,
		Other:     answer.Other,
		Pain:      answer.Pain,
		Org:       answer.Org,
		Notify:    answer.Notify,
		Source:    answer.Source,
		UpdatedAt: time.Now().UTC(),
	}
	return g.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "lms_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "feature", "other", "pain", "org", "notify", "source", "updated_at"}),
	}).Create(&row).Error
}

// GetFeatureInterest returns the account's answer, or gorm.ErrRecordNotFound.
func (g *LicensingGatewayImpl) GetFeatureInterest(lmsUserID string) (*models.FeatureInterestCore, error) {
	var row models.FeatureInterestDB
	if err := g.db.Where("lms_user_id = ?", lmsUserID).First(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// ListFeatureInterest returns a page of answers, newest first, and the total count.
func (g *LicensingGatewayImpl) ListFeatureInterest(limit, offset int) ([]models.FeatureInterestCore, int64, error) {
	var total int64
	if err := g.db.Model(&models.FeatureInterestDB{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.FeatureInterestDB
	if err := g.db.Order("updated_at DESC, lms_user_id").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]models.FeatureInterestCore, 0, len(rows))
	for i := range rows {
		out = append(out, *rows[i].ToCore())
	}
	return out, total, nil
}
