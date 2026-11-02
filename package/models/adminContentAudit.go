package models

import "time"

// Admin content audit actions.
const (
	AuditProjectDownload = "project_download"
	AuditProjectDelete   = "project_delete"
	AuditFileDownload    = "file_download"
	AuditFileDelete      = "file_delete"
)

// AdminContentAuditDB maps admin_content_audit: one super-admin action on someone's content.
type AdminContentAuditDB struct {
	ID           string    `gorm:"column:id;type:uuid;primaryKey"`
	ActorUserID  string    `gorm:"column:actor_user_id"`
	TargetUserID string    `gorm:"column:target_user_id"`
	Action       string    `gorm:"column:action"`
	ObjectType   string    `gorm:"column:object_type"`
	ObjectID     string    `gorm:"column:object_id"`
	ObjectName   string    `gorm:"column:object_name"`
	SizeBytes    int64     `gorm:"column:size_bytes"`
	Reason       string    `gorm:"column:reason"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (AdminContentAuditDB) TableName() string {
	return "admin_content_audit"
}
