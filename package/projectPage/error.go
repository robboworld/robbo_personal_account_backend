package projectPage

import "errors"

var (
	ErrPageNotFound        = errors.New("page not found")
	ErrSb3ArchiveNotFound  = errors.New("project file not found")
	ErrInternalServerLevel = errors.New("internal server error")
	ErrBadRequest          = errors.New("bad request")
	ErrBadRequestBody      = errors.New("bad request body")
	ErrProjectLimitReached = errors.New("PROJECT_LIMIT_REACHED")
	ErrProjectSizeExceeded = errors.New("PROJECT_SIZE_EXCEEDED")
	// ErrInvalidProjectFile — project.json / .sb3 не проходит проверку Scratch (Stage + meta.semver).
	ErrInvalidProjectFile = errors.New("INVALID_PROJECT_FILE")
	// ErrCloudQuotaExceeded is kept as an alias for older clients.
	ErrCloudQuotaExceeded = ErrProjectSizeExceeded
)
