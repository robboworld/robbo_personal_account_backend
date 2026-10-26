package models

import "time"

// FeatureInterestDB is the GORM model for feature_interest (Licensing PG): the paid-edition
// feature a user needs first, one row per account.
type FeatureInterestDB struct {
	LmsUserID string    `gorm:"column:lms_user_id;primaryKey"`
	Role      string    `gorm:"column:role;not null"`
	Feature   string    `gorm:"column:feature;not null"`
	Other     string    `gorm:"column:other;not null;default:''"`
	Pain      string    `gorm:"column:pain;not null;default:''"`
	Org       string    `gorm:"column:org;not null;default:''"`
	Notify    bool      `gorm:"column:notify;not null;default:false"`
	Source    string    `gorm:"column:source;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (FeatureInterestDB) TableName() string { return "feature_interest" }

// FeatureInterestCore is the domain model used across layers.
type FeatureInterestCore struct {
	LmsUserID string
	Role      string
	Feature   string
	Other     string
	Pain      string
	Org       string
	Notify    bool
	Source    string
	UpdatedAt time.Time
}

func (db *FeatureInterestDB) ToCore() *FeatureInterestCore {
	return &FeatureInterestCore{
		LmsUserID: db.LmsUserID,
		Role:      db.Role,
		Feature:   db.Feature,
		Other:     db.Other,
		Pain:      db.Pain,
		Org:       db.Org,
		Notify:    db.Notify,
		Source:    db.Source,
		UpdatedAt: db.UpdatedAt,
	}
}
