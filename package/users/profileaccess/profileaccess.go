// Package profileaccess decides whose LMS profile a signed-in user may read by ID.
package profileaccess

import (
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// TeacherStudentReader reports whether studentID is in one of teacherID's active classes.
// Set from teacherclass wiring; nil means no teacher sees anyone else's profile.
var TeacherStudentReader func(teacherID, studentID string) bool

// CanRead: a SuperAdmin reads any profile, everyone reads their own, and a teacher also
// reads the students of their classes. Profile IDs are sequential, so anything wider lets
// a caller walk every LMS account (emails, names, birth years of children).
func CanRead(viewerID string, viewerRole models.Role, targetID string) bool {
	viewerID = strings.TrimSpace(viewerID)
	targetID = strings.TrimSpace(targetID)
	if viewerRole == models.SuperAdmin {
		return true
	}
	if viewerID == "" || viewerID == "0" || targetID == "" {
		return false
	}
	if viewerID == targetID {
		return true
	}
	if viewerRole == models.Teacher && TeacherStudentReader != nil {
		return TeacherStudentReader(viewerID, targetID)
	}
	return false
}
