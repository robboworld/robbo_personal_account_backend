package models

import "time"

type ScratchProjectDB struct {
	ID               string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	OwnerUserID      string     `gorm:"column:owner_user_id"`
	Title            string     `gorm:"column:title"`
	Instruction      string     `gorm:"column:instruction"`
	Note             string     `gorm:"column:note"`
	ScratchVMJSON    string     `gorm:"column:scratch_vm_json;type:text"`
	IsPublic         bool       `gorm:"column:is_public"`
	LandingFeatured  bool       `gorm:"column:landing_featured"`
	PreviewImage     []byte     `gorm:"column:preview_image"`
	PreviewMime      string     `gorm:"column:preview_mime"`
	PreviewUpdatedAt *time.Time `gorm:"column:preview_updated_at"`
	VersionCounter   int64      `gorm:"column:version_counter"`
	CurrentVersionID *string    `gorm:"column:current_version_id"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
}

func (ScratchProjectDB) TableName() string {
	return "scratch_projects"
}
