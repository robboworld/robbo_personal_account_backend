package models

import "time"

// ScratchProjectVersionDB maps scratch_project_versions (.sb3 BYTEA storage).
type ScratchProjectVersionDB struct {
	ID              string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID       string    `gorm:"column:project_id"`
	VersionSeq      int64     `gorm:"column:version_seq"`
	Archive         []byte    `gorm:"column:archive"`
	SizeBytes       int64     `gorm:"column:size_bytes"`
	ChecksumSHA256  []byte    `gorm:"column:checksum_sha256"`
	CreatedByUserID string    `gorm:"column:created_by_user_id"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	SaveSource      *string   `gorm:"column:save_source"`
}

func (ScratchProjectVersionDB) TableName() string {
	return "scratch_project_versions"
}
