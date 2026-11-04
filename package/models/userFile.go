package models

import "time"

// UserFileDB maps user_files ("Мои файлы"); the bytes live in the userfiles blob store under ID.
type UserFileDB struct {
	ID           string `gorm:"column:id;type:uuid;primaryKey"`
	OwnerUserID  string `gorm:"column:owner_user_id"`
	OriginalName string `gorm:"column:original_name"`
	Kind         string `gorm:"column:kind"`
	Mime         string `gorm:"column:mime"`
	SizeBytes    int64  `gorm:"column:size_bytes"`
	SHA256       string `gorm:"column:sha256"`
	HasScripts   bool   `gorm:"column:has_scripts"` // Python scripts inside a zip or FreeCAD model
	// ScannedAt: when ClamAV found the file clean; nil if it was never scanned.
	ScannedAt *time.Time `gorm:"column:scanned_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (UserFileDB) TableName() string {
	return "user_files"
}

// UserFileHTTP is a file as the ЛК lists it.
type UserFileHTTP struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Mime       string `json:"mime"`
	SizeBytes  int64  `json:"sizeBytes"`
	CreatedAt  string `json:"createdAt"`
	HasScripts bool   `json:"hasScripts"`
}

func (f *UserFileDB) ToHTTP() UserFileHTTP {
	return UserFileHTTP{
		ID:         f.ID,
		Name:       f.OriginalName,
		Kind:       f.Kind,
		Mime:       f.Mime,
		SizeBytes:  f.SizeBytes,
		CreatedAt:  f.CreatedAt.UTC().Format(time.RFC3339),
		HasScripts: f.HasScripts,
	}
}
