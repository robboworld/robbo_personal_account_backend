package models

// ScratchProjectVersionDB maps scratch_project_versions (.sb3 BYTEA storage).
type ScratchProjectVersionDB struct {
	ID         string `gorm:"column:id;primaryKey"`
	ProjectID  string `gorm:"column:project_id"`
	VersionSeq int64  `gorm:"column:version_seq"`
	Archive    []byte `gorm:"column:archive"`
}

func (ScratchProjectVersionDB) TableName() string {
	return "scratch_project_versions"
}
