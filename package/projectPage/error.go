package projectPage

import (
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
)

var (
	ErrPageNotFound        = errors.New("page not found")
	ErrSb3ArchiveNotFound  = errors.New("project file not found")
	ErrInternalServerLevel = errors.New("internal server error")
	ErrBadRequest          = errors.New("bad request")
	ErrBadRequestBody      = errors.New("bad request body")
	ErrProjectSizeExceeded = errors.New("PROJECT_SIZE_EXCEEDED")
	// ErrStorageQuotaExceeded — the account total is used up; only growth is refused.
	// The same value as storage.ErrStorageQuotaExceeded (callers compare with ==).
	ErrStorageQuotaExceeded = storage.ErrStorageQuotaExceeded
	// ErrInvalidProjectFile — project.json / .sb3 не проходит проверку Scratch (Stage + meta.semver).
	ErrInvalidProjectFile = errors.New("INVALID_PROJECT_FILE")
	// ErrCloudQuotaExceeded is kept as an alias for older clients.
	ErrCloudQuotaExceeded = ErrProjectSizeExceeded
	ErrProfanityDetected  = errors.New("profanity_detected")
	ErrCommentNotFound    = errors.New("comment not found")
)
