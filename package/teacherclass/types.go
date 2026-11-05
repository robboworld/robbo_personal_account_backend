package teacherclass

import (
	"errors"
	"io"
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
	// File assignments need "Мои файлы" and a required antivirus scan.
	ErrFileAssignmentsDisabled = errors.New("file_assignments_disabled")
	// The file's type is not one the assignment accepts.
	ErrFileKindNotAllowed = errors.New("file_kind_not_allowed")
	// The submission already has classes.maxSubmissionFiles files.
	ErrTooManySubmissionFiles = errors.New("too_many_submission_files")
	// A file assignment is handed in with at least one file.
	ErrNoSubmissionFiles = errors.New("no_submission_files")
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

// FileRefDTO is a file of "Мои файлы" as a class sees it: no owner, no download URL.
type FileRefDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	SizeBytes  int64  `json:"sizeBytes"`
	HasScripts bool   `json:"hasScripts"`
}

func FileRef(f *models.UserFileDB) FileRefDTO {
	return FileRefDTO{ID: f.ID, Name: f.OriginalName, Kind: f.Kind, SizeBytes: f.SizeBytes, HasScripts: f.HasScripts}
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
	// Kind is "scratch" or "file". A file assignment accepts AllowedKinds (empty: any kind
	// "Мои файлы" takes) and may come with the teacher's TemplateFile.
	Kind         string      `json:"kind"`
	AllowedKinds []string    `json:"allowedKinds"`
	TemplateFile *FileRefDTO `json:"templateFile,omitempty"`
	// Student view only: the student's own copy of the template and its review state.
	MyProjectID     string `json:"myProjectId,omitempty"`
	MyReviewStatus  string `json:"myReviewStatus,omitempty"`
	MyReviewComment string `json:"myReviewComment,omitempty"`
	// Student view of a file assignment: the submission and the files attached to it.
	MySubmissionID string       `json:"mySubmissionId,omitempty"`
	MyFiles        []FileRefDTO `json:"myFiles,omitempty"`
	// MaxFiles a submission may hold (file assignments).
	MaxFiles int `json:"maxFiles,omitempty"`
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
	// Kind is models.AssignmentScratch (default) or models.AssignmentFile.
	Kind           string
	AllowedKinds   []string
	TemplateFileID string
}

// SubmissionDTO is one student's copy. The teacher gets a draft without ProjectID and
// PreviewURL: when it was last saved, not what is in it.
type SubmissionDTO struct {
	ProjectID string `json:"projectId,omitempty"`
	// SubmissionID and Files are set for a file assignment; Files only for handed-in work.
	SubmissionID  string       `json:"submissionId,omitempty"`
	Files         []FileRefDTO `json:"files,omitempty"`
	OwnerUserID   string       `json:"ownerUserId"`
	Title         string       `json:"title"`
	ReviewStatus  string       `json:"reviewStatus"`
	ReviewComment string       `json:"reviewComment,omitempty"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	SubmittedAt   *time.Time   `json:"submittedAt,omitempty"`
	PreviewURL    string       `json:"previewUrl,omitempty"`
	// Late: handed in after the assignment's due date.
	Late bool `json:"late,omitempty"`
}

type ProgressCellDTO struct {
	AssignmentID string `json:"assignmentId"`
	Status       string `json:"status"` // empty|draft|submitted|returned|accepted|shown_in_class|missing
	ProjectID    string `json:"projectId,omitempty"`
	SubmissionID string `json:"submissionId,omitempty"`
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

	// File assignments.
	GetFileSubmission(id string) (*models.ClassFileSubmissionDB, error)
	FindFileSubmission(assignmentID, studentUserID string) (*models.ClassFileSubmissionDB, error)
	// EnsureFileSubmission returns the student's submission, creating a draft if there is none.
	EnsureFileSubmission(assignmentID, studentUserID string) (*models.ClassFileSubmissionDB, error)
	ListFileSubmissions(assignmentIDs []string) ([]models.ClassFileSubmissionDB, error)
	// ListSubmissionFiles returns the live files of each submission, oldest attached first.
	ListSubmissionFiles(submissionIDs []string) (map[string][]models.UserFileDB, error)
	// AttachSubmissionFile adds the file unless the submission already holds maxFiles
	// (ErrTooManySubmissionFiles); attaching it twice is a no-op.
	AttachSubmissionFile(submissionID, fileID string, maxFiles int) error
	DetachSubmissionFile(submissionID, fileID string) error
	UpdateFileReview(submissionID, status, comment string, submittedAt *time.Time) error
	// GetUserFiles returns the live "Мои файлы" rows among ids, by id.
	GetUserFiles(ids []string) (map[string]models.UserFileDB, error)
	// FileInUse: the file is handed in (not returned) or the starter file of an open
	// assignment, in a class that is not archived.
	FileInUse(ownerUserID, fileID string) (bool, error)
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
	// LeaveClass takes the student off the roster; projects and handed-in work stay.
	LeaveClass(studentID, classID string) error

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

	// File assignments.
	FileAssignmentsEnabled() bool
	AttachFile(studentID, assignmentID, fileID string) (*AssignmentDTO, error)
	DetachFile(studentID, assignmentID, fileID string) (*AssignmentDTO, error)
	SubmitFiles(studentID, assignmentID string) error
	// OpenStarterFile is the assignment's starter file for the class owner or a member.
	OpenStarterFile(viewerID string, role models.Role, assignmentID string) (*models.UserFileDB, io.ReadSeekCloser, error)
	// OpenSubmissionFile is a file of handed-in work for the class owner (any work for a SuperAdmin).
	OpenSubmissionFile(teacherID string, role models.Role, submissionID, fileID string) (*models.UserFileDB, io.ReadSeekCloser, error)
	ReviewFileSubmission(teacherID string, role models.Role, submissionID, status, comment string) error
}
