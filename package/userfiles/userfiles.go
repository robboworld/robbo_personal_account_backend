// Package userfiles is "Мои файлы": an account's own files in the ЛК, visible only to the
// owner and counted in the same storage quota as Scratch projects (package storage).
package userfiles

import (
	"errors"
	"io"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

var (
	// ErrDisabled — userFiles.enabled is off; the routes answer 404.
	ErrDisabled           = errors.New("user files are disabled")
	ErrFileNotFound       = errors.New("FILE_NOT_FOUND")
	ErrFileTypeNotAllowed = errors.New("FILE_TYPE_NOT_ALLOWED")
	ErrFileSizeExceeded   = errors.New("FILE_SIZE_EXCEEDED")
	ErrFileCountExceeded  = errors.New("FILE_COUNT_EXCEEDED")
	// ErrFileContentNotAllowed — the type is allowed but the contents are not: macros,
	// scripts, executables or archives inside, an image too large to decode, an encrypted PDF.
	ErrFileContentNotAllowed = errors.New("FILE_CONTENT_NOT_ALLOWED")
	// ErrTooManyUploads — the account already has userFiles.maxConcurrentUploads in progress.
	ErrTooManyUploads = errors.New("TOO_MANY_UPLOADS")
	// ErrUploadRateLimited — userFiles.maxUploadsPerHour reached (deleted files count too).
	ErrUploadRateLimited = errors.New("UPLOAD_RATE_LIMITED")
	// ErrStorageUnavailable — the blob volume is short of free space.
	ErrStorageUnavailable = errors.New("STORAGE_UNAVAILABLE")
)

type Gateway interface {
	// GetTotalStorageBytesForOwner sums the owner's files that are not deleted (storage.UsageSource).
	GetTotalStorageBytesForOwner(ownerUserID string) (int64, error)
	CountForOwner(ownerUserID string) (int64, error)
	// CountCreatedSince counts uploads since t, deleted ones included.
	CountCreatedSince(ownerUserID string, t time.Time) (int64, error)
	ListByOwner(ownerUserID string) ([]*models.UserFileDB, error)
	// GetByOwnerAndID returns ErrFileNotFound for a deleted file or someone else's.
	GetByOwnerAndID(ownerUserID, id string) (*models.UserFileDB, error)
	// CreateLocked inserts row after check passes, both under one per-owner lock, so two
	// uploads at once cannot both fit into the last free megabytes.
	CreateLocked(row *models.UserFileDB, check func() error) error
	SoftDelete(ownerUserID, id string) error
	ListDeletedBefore(t time.Time, limit int) ([]string, error)
	HardDelete(ids []string) error
}

type UseCase interface {
	Enabled() bool
	MaxUploadBytes(ownerUserID string) (int64, error)
	Upload(ownerUserID, name string, r io.Reader) (*models.UserFileDB, error)
	List(ownerUserID string) ([]*models.UserFileDB, error)
	CountFiles(ownerUserID string) (int64, error)
	Open(ownerUserID, id string) (*models.UserFileDB, io.ReadSeekCloser, error)
	Delete(ownerUserID, id string) error
	// CollectGarbage removes rows of files deleted longer ago than the retention, and
	// leftovers of interrupted uploads. The bytes go at Delete already.
	CollectGarbage(now time.Time) (removed int, err error)
}
