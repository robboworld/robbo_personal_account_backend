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

// CohortTeacherReader optionally grants read access to class homework projects.
// Set from teacherclass wiring: (teacherID, courseID, cohortID) -> owns class.
var CohortTeacherReader func(teacherID, courseID, cohortID string) bool

func normalizeUserID(id string) string {
	return strings.TrimSpace(id)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Resolve returns access for viewerUserID against project ownership and visibility.
func Resolve(viewerUserID string, p *models.ScratchProjectDB) ProjectAccess {
	isOwner := normalizeUserID(p.OwnerUserID) == normalizeUserID(viewerUserID)
	canRead := isOwner || p.IsPublic
	// The class teacher reads homework once it is handed in; drafts stay the student's.
	if !canRead && CohortTeacherReader != nil && p.EdxCohortID != nil && strings.TrimSpace(*p.EdxCohortID) != "" &&
		models.IsHandedIn(p.ReviewStatus) {
		canRead = CohortTeacherReader(viewerUserID, derefStr(p.EdxCourseID), *p.EdxCohortID)
	}
	return ProjectAccess{
		IsOwner:  isOwner,
		CanRead:  canRead,
		CanWrite: isOwner,
	}
}

// SuperAdminReadsClassWork mirrors teacherclass canManage: a SuperAdmin manages every class,
// so it reads class homework (projects keyed to a cohort). Other private projects stay closed.
func SuperAdminReadsClassWork(role models.Role, p *models.ScratchProjectDB) bool {
	return role == models.SuperAdmin && p.EdxCohortID != nil && strings.TrimSpace(*p.EdxCohortID) != ""
}
