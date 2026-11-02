package models

import "time"

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
}

func (ClassAssignmentDB) TableName() string { return "class_assignments" }

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
