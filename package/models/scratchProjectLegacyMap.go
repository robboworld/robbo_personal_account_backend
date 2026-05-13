package models

import "time"

type ScratchProjectLegacyMapDB struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	LegacyProjectID     *string   `gorm:"column:legacy_project_id"`
	LegacyProjectPageID *string   `gorm:"column:legacy_project_page_id"`
	StorageProjectID    string    `gorm:"column:storage_project_id"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

func (ScratchProjectLegacyMapDB) TableName() string {
	return "scratch_project_legacy_map"
}
