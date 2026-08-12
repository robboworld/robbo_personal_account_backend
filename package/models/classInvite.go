package models

import "time"

type ClassInviteDB struct {
	ID              string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	EdxCourseID     string     `gorm:"column:edx_course_id"`
	EdxCohortID     string     `gorm:"column:edx_cohort_id"`
	OwnerTeacherID  string     `gorm:"column:owner_teacher_id"`
	DisplayName     string     `gorm:"column:display_name"`
	InviteCode      string     `gorm:"column:invite_code"`
	InviteSlug      string     `gorm:"column:invite_slug"`
	IsActive        bool       `gorm:"column:is_active"`
	RotatedAt       *time.Time `gorm:"column:rotated_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	ArchivedAt      *time.Time `gorm:"column:archived_at"`
}

func (ClassInviteDB) TableName() string { return "class_invites" }

type ClassAssignmentDB struct {
	ID                  string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	EdxCourseID         string     `gorm:"column:edx_course_id"`
	EdxCohortID         string     `gorm:"column:edx_cohort_id"`
	CreatedByTeacherID  string     `gorm:"column:created_by_teacher_id"`
	Title               string     `gorm:"column:title"`
	Instructions        string     `gorm:"column:instructions"`
	AssignmentKey       string     `gorm:"column:assignment_key"`
	TemplateProjectID   *string    `gorm:"column:template_project_id;type:uuid"`
	DueAt               *time.Time `gorm:"column:due_at"`
	ClosedAt            *time.Time `gorm:"column:closed_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
}

func (ClassAssignmentDB) TableName() string { return "class_assignments" }

// Class review statuses on scratch_projects.review_status
const (
	ReviewDraft         = "draft"
	ReviewSubmitted     = "submitted"
	ReviewReturned      = "returned"
	ReviewAccepted      = "accepted"
	ReviewShownInClass  = "shown_in_class"
)
