package models

import "time"

// ClassGalleryItemDB is a file shared with the class in the gallery of one assignment (a lesson).
// It links a file in the uploader's "Мои файлы": the file is not copied and stays in their quota.
type ClassGalleryItemDB struct {
	ID             string `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	AssignmentID   string `gorm:"column:assignment_id;type:uuid"`
	UserFileID     string `gorm:"column:user_file_id;type:uuid"`
	UploaderUserID string `gorm:"column:uploader_user_id"`
	// Caption, Pinned and Position are set by the class's teachers.
	Caption   string    `gorm:"column:caption"`
	Pinned    bool      `gorm:"column:pinned"`
	Position  int       `gorm:"column:position"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ClassGalleryItemDB) TableName() string { return "class_gallery_items" }
