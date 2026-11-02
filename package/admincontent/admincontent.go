// Package admincontent lets a super admin see, download and delete other users' projects and
// files ("Мои файлы") with a reason, every action in the admin audit log, and watch the
// instance's storage.
package admincontent

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("NOT_FOUND")

// ProjectItem is a project as the moderator lists it, private ones included.
type ProjectItem struct {
	ID        string    `gorm:"column:id"`
	Title     string    `gorm:"column:title"`
	IsPublic  bool      `gorm:"column:is_public"`
	SizeBytes int64     `gorm:"column:size_bytes"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// OwnerUsage is one account's stored bytes.
type OwnerUsage struct {
	OwnerUserID   string `gorm:"column:owner"`
	ProjectsBytes int64  `gorm:"column:projects_bytes"`
	FilesBytes    int64  `gorm:"column:files_bytes"`
}

// Totals is what the whole service stores.
type Totals struct {
	ProjectsBytes  int64 `gorm:"column:projects_bytes" json:"projectsBytes"`
	ProjectCount   int64 `gorm:"column:project_count" json:"projectCount"`
	FilesBytes     int64 `gorm:"column:files_bytes" json:"filesBytes"`
	FileCount      int64 `gorm:"column:file_count" json:"fileCount"`
	OwnersWithData int64 `gorm:"column:owners_with_data" json:"ownersWithData"`
}

type Gateway interface {
	ListProjectsForOwner(ownerUserID string, limit int) ([]ProjectItem, error)
	StorageTotals() (Totals, error)
	TopOwners(limit int) ([]OwnerUsage, error)
	ProjectsDatabaseBytes() (int64, error)
	LicensingDatabaseBytes() (int64, error)
}

// Volume reports the disk under the user files store.
type Volume interface {
	TotalBytes() (int64, error)
	FreeBytes() (int64, error)
}
