package models

import (
	"time"

	"github.com/lib/pq"
)

type ClassInviteDB struct {
	ID             string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	EdxCourseID    string     `gorm:"column:edx_course_id"`
	EdxCohortID    string     `gorm:"column:edx_cohort_id"`
	OwnerTeacherID string     `gorm:"column:owner_teacher_id"`
	DisplayName    string     `gorm:"column:display_name"`
	InviteCode     string     `gorm:"column:invite_code"`
	InviteSlug     string     `gorm:"column:invite_slug"`
	IsActive       bool       `gorm:"column:is_active"`
	RotatedAt      *time.Time `gorm:"column:rotated_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
	ArchivedAt     *time.Time `gorm:"column:archived_at"`
	// LmsLinked: the class is an Open edX cohort. When false the roster is class_members,
	// EdxCourseID is empty and EdxCohortID is UnlinkedCohortKey(ID).
	LmsLinked bool `gorm:"column:lms_linked"`
}

func (ClassInviteDB) TableName() string { return "class_invites" }

// UnlinkedCohortKey stands in for the cohort id of a class kept only in the ЛК, so
// assignments and homework projects key on (course, cohort) for every class alike.
func UnlinkedCohortKey(classID string) string { return "lk:" + classID }

type ClassMemberDB struct {
	ClassID  string    `gorm:"type:uuid;primaryKey;column:class_id"`
	UserID   string    `gorm:"primaryKey;column:user_id"`
	JoinedAt time.Time `gorm:"column:joined_at"`
}

func (ClassMemberDB) TableName() string { return "class_members" }

// ClassCoTeacherDB is one term of a teacher running a class next to its owner. It counts while
// RevokedAt is nil and ExpiresAt is ahead.
type ClassCoTeacherDB struct {
	ID         string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	ClassID    string     `gorm:"column:class_id;type:uuid"`
	TeacherID  string     `gorm:"column:teacher_id"`
	GrantedBy  string     `gorm:"column:granted_by"`
	GrantedAt  time.Time  `gorm:"column:granted_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	RevokedBy  *string    `gorm:"column:revoked_by"`
	RemindedAt *time.Time `gorm:"column:reminded_at"`
}

func (ClassCoTeacherDB) TableName() string { return "class_co_teachers" }

type ClassAssignmentDB struct {
	ID                 string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	EdxCourseID        string     `gorm:"column:edx_course_id"`
	EdxCohortID        string     `gorm:"column:edx_cohort_id"`
	CreatedByTeacherID string     `gorm:"column:created_by_teacher_id"`
	Title              string     `gorm:"column:title"`
	Instructions       string     `gorm:"column:instructions"`
	AssignmentKey      string     `gorm:"column:assignment_key"`
	TemplateProjectID  *string    `gorm:"column:template_project_id;type:uuid"`
	DueAt              *time.Time `gorm:"column:due_at"`
	ClosedAt           *time.Time `gorm:"column:closed_at"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
	// Kind is AssignmentScratch or AssignmentFile; an empty value reads as Scratch.
	Kind string `gorm:"column:kind;default:scratch"`
	// AllowedKinds are user_files.kind values a file assignment accepts; empty means any.
	AllowedKinds pq.StringArray `gorm:"column:allowed_kinds;type:text[];not null;default:'{}'"`
	// TemplateFileID is the teacher's optional starter file for a file assignment.
	TemplateFileID *string `gorm:"column:template_file_id;type:uuid"`
}

func (ClassAssignmentDB) TableName() string { return "class_assignments" }

const (
	// AssignmentScratch: every student gets a copy of the template project (or an empty one).
	AssignmentScratch = "scratch"
	// AssignmentFile: students hand in files from "Мои файлы" made in any program.
	AssignmentFile = "file"
)

// IsFileAssignment: students hand in files, not a Scratch project.
func (a *ClassAssignmentDB) IsFileAssignment() bool { return a.Kind == AssignmentFile }

// ClassFileSubmissionDB is one student's work on a file assignment.
type ClassFileSubmissionDB struct {
	ID            string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	AssignmentID  string     `gorm:"column:assignment_id;type:uuid"`
	StudentUserID string     `gorm:"column:student_user_id"`
	ReviewStatus  string     `gorm:"column:review_status"`
	ReviewComment string     `gorm:"column:review_comment"`
	SubmittedAt   *time.Time `gorm:"column:submitted_at"`
	// SubmissionNote is what the student wrote to the teacher at the last hand-in.
	SubmissionNote string `gorm:"column:submission_note"`
	// ReviewedByTeacherID and ReviewedAt: who gave the current review; a new hand-in clears them.
	ReviewedByTeacherID *string    `gorm:"column:reviewed_by_teacher_id"`
	ReviewedAt          *time.Time `gorm:"column:reviewed_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
}

func (ClassFileSubmissionDB) TableName() string { return "class_file_submissions" }

// ClassSubmissionFileDB links a file in the student's "Мои файлы" to their submission.
type ClassSubmissionFileDB struct {
	SubmissionID string    `gorm:"primaryKey;column:submission_id;type:uuid"`
	UserFileID   string    `gorm:"primaryKey;column:user_file_id;type:uuid"`
	AddedAt      time.Time `gorm:"column:added_at"`
}

func (ClassSubmissionFileDB) TableName() string { return "class_submission_files" }

// Class review statuses on scratch_projects.review_status
const (
	ReviewDraft        = "draft"
	ReviewSubmitted    = "submitted"
	ReviewReturned     = "returned"
	ReviewAccepted     = "accepted"
	ReviewShownInClass = "shown_in_class"
)

// IsHandedIn: the student submitted the work (it may be reviewed since). A draft (or no
// status) is the student's private work: only its owner and a SuperAdmin see it.
func IsHandedIn(status *string) bool {
	if status == nil {
		return false
	}
	switch *status {
	case ReviewSubmitted, ReviewReturned, ReviewAccepted, ReviewShownInClass:
		return true
	}
	return false
}

// IsReviewLocked: the work is with the teacher (handed in and not returned for rework). Its
// owner may not change it, so the teacher reviews exactly what was handed in.
func IsReviewLocked(status *string) bool {
	if status == nil {
		return false
	}
	switch *status {
	case ReviewSubmitted, ReviewAccepted, ReviewShownInClass:
		return true
	}
	return false
}
