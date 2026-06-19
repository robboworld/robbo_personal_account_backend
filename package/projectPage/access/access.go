package access

import (
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// ProjectAccess describes read/write rights for a scratch project row.
type ProjectAccess struct {
	CanRead  bool
	CanWrite bool
	IsOwner  bool
}

func normalizeUserID(id string) string {
	return strings.TrimSpace(id)
}

// Resolve returns access for viewerUserID against project ownership and visibility.
func Resolve(viewerUserID string, p *models.ScratchProjectDB) ProjectAccess {
	isOwner := normalizeUserID(p.OwnerUserID) == normalizeUserID(viewerUserID)
	return ProjectAccess{
		IsOwner:  isOwner,
		CanRead:  isOwner || p.IsPublic,
		CanWrite: isOwner,
	}
}
