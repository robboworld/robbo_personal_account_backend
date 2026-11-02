package teacherclass

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

var (
	ErrBadRequest   = errors.New("bad request")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal error")
	ErrAlreadyIn    = errors.New("already in class")
	// The teacher stopped accepting work for the assignment.
	ErrAssignmentClosed = errors.New("assignment_closed")
	// The work is on review or reviewed: only a draft or returned work is handed in.
	ErrAlreadyHandedIn = errors.New("already_handed_in")
)

type CreateClassInput struct {
	CourseID    string
	DisplayName string
	// LinkLMS makes the class an Open edX cohort on CourseID; otherwise the ЛК keeps the roster.
	LinkLMS bool
}

type ClassDTO struct {
	ID             string    `json:"id"`
	EdxCourseID    string    `json:"edxCourseId"`
	EdxCohortID    string    `json:"edxCohortId"`
	OwnerTeacherID string    `json:"ownerTeacherId"`
	DisplayName    string    `json:"displayName"`
	InviteCode     string    `json:"inviteCode"`
	InviteSlug     string    `json:"inviteSlug"`
	JoinURL        string    `json:"joinUrl"`
	IsActive       bool      `json:"isActive"`
	LmsLinked      bool      `json:"lmsLinked"`
	CreatedAt      time.Time `json:"createdAt"`
	OwnerName      string    `json:"ownerName,omitempty"`
}

// CourseDTO is a course a class can be created on.
type CourseDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JoinPreviewDTO struct {
	DisplayName string `json:"displayName"`
	EdxCourseID string `json:"edxCourseId"`
	InviteSlug  string `json:"inviteSlug"`
	OwnerName   string `json:"ownerName,omitempty"`
	IsActive    bool   `json:"isActive"`
	LmsLinked   bool   `json:"lmsLinked"`
}

type CohortUserDTO struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	UserID   string `json:"userId,omitempty"`
	Name     string `json:"name,omitempty"`
}

type AssignmentDTO struct {
	ID                 string     `json:"id"`
	EdxCourseID        string     `json:"edxCourseId"`
	EdxCohortID        string     `json:"edxCohortId"`
	CreatedByTeacherID string     `json:"createdByTeacherId"`
	Title              string     `json:"title"`
	Instructions       string     `json:"instructions"`
	AssignmentKey      string     `json:"assignmentKey"`
	TemplateProjectID  *string    `json:"templateProjectId,omitempty"`
	DueAt              *time.Time `json:"dueAt,omitempty"`
	ClosedAt           *time.Time `json:"closedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	// Student view only: the student's own copy of the template and its review state.
	MyProjectID     string `json:"myProjectId,omitempty"`
	MyReviewStatus  string `json:"myReviewStatus,omitempty"`
	MyReviewComment string `json:"myReviewComment,omitempty"`
	// CreateAssignment only: copies made, failed (issue-late retries them) and skipped
	// (a cohort member without an LMS user id cannot own a project).
	CopiesIssued  int `json:"copiesIssued,omitempty"`
	CopiesFailed  int `json:"copiesFailed,omitempty"`
	CopiesSkipped int `json:"copiesSkipped,omitempty"`
}

type CreateAssignmentInput struct {
	Title             string
	Instructions      string
	AssignmentKey     string
	TemplateProjectID string
	DueAt             *time.Time
}

// SubmissionDTO is one student's copy. The teacher gets a draft without ProjectID and
// PreviewURL: when it was last saved, not what is in it.
type SubmissionDTO struct {
	ProjectID     string     `json:"projectId,omitempty"`
	OwnerUserID   string     `json:"ownerUserId"`
	Title         string     `json:"title"`
	ReviewStatus  string     `json:"reviewStatus"`
	ReviewComment string     `json:"reviewComment,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	SubmittedAt   *time.Time `json:"submittedAt,omitempty"`
	PreviewURL    string     `json:"previewUrl,omitempty"`
	// Late: handed in after the assignment's due date.
	Late bool `json:"late,omitempty"`
}

type ProgressCellDTO struct {
	AssignmentID string `json:"assignmentId"`
	Status       string `json:"status"` // empty|draft|submitted|returned|accepted|shown_in_class|missing
	ProjectID    string `json:"projectId,omitempty"`
	Late         bool   `json:"late,omitempty"`
}

// ProgressRowDTO holds one project owner's cells; assignments without a project are omitted.
type ProgressRowDTO struct {
	UserID string            `json:"userId"`
	Cells  []ProgressCellDTO `json:"cells"`
}

type Gateway interface {
	CreateInvite(inv *models.ClassInviteDB) error
	UpdateInvite(inv *models.ClassInviteDB) error
	GetInviteByID(id string) (*models.ClassInviteDB, error)
	GetInviteBySlug(slug string) (*models.ClassInviteDB, error)
	GetInviteByCode(code string) (*models.ClassInviteDB, error)
	GetInviteByCourseCohort(courseID, cohortID string) (*models.ClassInviteDB, error)
	ListInvitesByOwner(ownerTeacherID string) ([]models.ClassInviteDB, error)
	ListAllInvites() ([]models.ClassInviteDB, error)
	ListActiveInvites() ([]models.ClassInviteDB, error)

	// Roster of classes not linked to the LMS.
	AddMember(classID, userID string) error
	RemoveMember(classID, userID string) error
	ListMembers(classID string) ([]models.ClassMemberDB, error)
	ListMemberClassIDs(userID string) ([]string, error)
	IsMemberOfAny(userID string, classIDs []string) (bool, error)

	CreateAssignment(a *models.ClassAssignmentDB) error
	ListAssignments(courseID, cohortID string) ([]models.ClassAssignmentDB, error)
	GetAssignment(id string) (*models.ClassAssignmentDB, error)
	SetAssignmentClosed(id string, closed bool) error

	GetProject(id string) (*models.ScratchProjectDB, error)
	// ForkProjectForStudent with a nil template issues an empty project.
	ForkProjectForStudent(template *models.ScratchProjectDB, studentUserID, title string, assignment *models.ClassAssignmentDB) (*models.ScratchProjectDB, error)
	ListSubmissions(assignmentID string) ([]models.ScratchProjectDB, error)
	ListSubmissionsForAssignments(assignmentIDs []string) ([]models.ScratchProjectDB, error)
	ListProjectsByCohortAssignment(cohortID, assignmentID string) ([]models.ScratchProjectDB, error)
	UpdateReview(projectID, status, comment string, submittedAt *time.Time) error
	FindStudentAssignmentProject(assignmentID, studentUserID string) (*models.ScratchProjectDB, error)
}

type UseCase interface {
	CreateClass(teacherID string, role models.Role, in CreateClassInput) (*ClassDTO, error)
	ListMyClasses(teacherID string, role models.Role) ([]ClassDTO, error)
	ListCourses(teacherID string, role models.Role) ([]CourseDTO, error)
	GetClass(teacherID string, role models.Role, classID string) (*ClassDTO, error)
	RenameClass(teacherID string, role models.Role, classID, displayName string) (*ClassDTO, error)
	ArchiveClass(teacherID string, role models.Role, classID string) error
	RotateInvite(teacherID string, role models.Role, classID string) (*ClassDTO, error)

	PreviewJoin(codeOrSlug string) (*JoinPreviewDTO, error)
	JoinClass(studentID string, codeOrSlug string) (*ClassDTO, error)
	ListStudentClasses(studentID string) ([]ClassDTO, error)

	ListMembers(teacherID string, role models.Role, classID string) ([]CohortUserDTO, error)
	AddMembersByEmail(teacherID string, role models.Role, classID string, emails []string) (added []string, missing []string, err error)
	RemoveMember(teacherID string, role models.Role, classID, username string) error

	CreateAssignment(teacherID string, role models.Role, classID string, in CreateAssignmentInput) (*AssignmentDTO, error)
	ListAssignments(viewerID string, role models.Role, classID string) ([]AssignmentDTO, error)
	IssueToLatecomers(teacherID string, role models.Role, classID, assignmentID string) (issued int, err error)
	SetAssignmentClosed(teacherID string, role models.Role, classID, assignmentID string, closed bool) (*AssignmentDTO, error)
	SubmitAssignment(studentID, projectID string) error
	ReviewSubmission(teacherID string, role models.Role, projectID, status, comment string) error
	ListSubmissions(teacherID string, role models.Role, classID, assignmentID string) ([]SubmissionDTO, error)
	ProgressMatrix(teacherID string, role models.Role, classID string) (assignments []AssignmentDTO, rows []ProgressRowDTO, err error)
}
