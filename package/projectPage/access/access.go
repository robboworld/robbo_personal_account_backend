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

// HomeworkLocked: an assignment copy that is handed in and not returned. Its owner may read it
// but not change it (upload, preview, fields, delete) until the teacher returns it.
func HomeworkLocked(p *models.ScratchProjectDB) bool {
	return p.AssignmentID != nil && strings.TrimSpace(*p.AssignmentID) != "" && models.IsReviewLocked(p.ReviewStatus)
}

// RunsClass: the viewer is the owner or a co-teacher of the class the project was issued in.
func RunsClass(viewerUserID string, p *models.ScratchProjectDB) bool {
	if CohortTeacherReader == nil || normalizeUserID(viewerUserID) == "" {
		return false
	}
	if p.EdxCohortID == nil || strings.TrimSpace(*p.EdxCohortID) == "" {
		return false
	}
	return CohortTeacherReader(viewerUserID, derefStr(p.EdxCourseID), *p.EdxCohortID)
}

// TeacherCanEdit: who runs the class may change work only while it is with them. Its owner cannot
// save then (HomeworkLocked), so the two never overwrite each other. Returned work is the
// student's again. A SuperAdmin reads class work but does not change a child's work.
func TeacherCanEdit(viewerUserID string, p *models.ScratchProjectDB) bool {
	return HomeworkLocked(p) &&
		normalizeUserID(p.OwnerUserID) != normalizeUserID(viewerUserID) &&
		RunsClass(viewerUserID, p)
}

// EditingRole: who may save handed-in work now counts as editing it, "" for anyone else. A student
// is counted on returned work only: a draft stays private, locked work cannot be saved.
func EditingRole(viewerUserID string, p *models.ScratchProjectDB) string {
	if TeacherCanEdit(viewerUserID, p) {
		return models.EditorTeacher
	}
	if p.AssignmentID != nil && p.ReviewStatus != nil && *p.ReviewStatus == models.ReviewReturned &&
		normalizeUserID(p.OwnerUserID) == normalizeUserID(viewerUserID) {
		return models.EditorStudent
	}
	return ""
}

// SuperAdminReadsClassWork mirrors teacherclass canManage: a SuperAdmin manages every class,
// so it reads class homework (projects keyed to a cohort). Other private projects stay closed.
func SuperAdminReadsClassWork(role models.Role, p *models.ScratchProjectDB) bool {
	return role == models.SuperAdmin && p.EdxCohortID != nil && strings.TrimSpace(*p.EdxCohortID) != ""
}
