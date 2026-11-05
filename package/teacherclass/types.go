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
	// The student's note changes only while the work waits for review.
	ErrNotOnReview = errors.New("work_not_on_review")
	// The Scratch copy is still what the student was given: there is no work to hand in.
	ErrWorkNotEdited = errors.New("work_not_edited")
	// One error for "no such user" and "not a teacher": the form must not tell who is a teacher.
	ErrNotATeacher = errors.New("not_a_teacher")
	// The teacher already runs the class: its owner or a co-teacher with a running term.
	ErrAlreadyCoTeacher = errors.New("already_co_teacher")
	// The class already has classes.maxCoTeachers co-teachers.
	ErrTooManyCoTeachers = errors.New("too_many_co_teachers")
	// The term ends in the past or further than classes.coTeacherMaxDays ahead.
	ErrBadCoTeacherTerm = errors.New("bad_co_teacher_term")
	// A co-teacher knows the invite code: it must not let them back in as a student.
	ErrCoTeacherCannotJoin = errors.New("co_teacher_cannot_join")
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
	// CoTeacherUntil is set when the viewer runs the class as a co-teacher: the end of their term.
	CoTeacherUntil *time.Time `json:"coTeacherUntil,omitempty"`
}

// CoTeacherDTO is a co-teacher with a running term. The email stays with the owner who typed it.
type CoTeacherDTO struct {
	TeacherID string    `json:"teacherId"`
	Name      string    `json:"name,omitempty"`
	GrantedAt time.Time `json:"grantedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
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
	// MySubmissionNote is what the student wrote at the last hand-in, of either kind of work.
	MySubmissionNote string `json:"mySubmissionNote,omitempty"`
	// MyEditedAt and its name: the teacher changed the student's Scratch work (until the next hand-in).
	MyEditedAt     *time.Time `json:"myEditedAt,omitempty"`
	MyEditedByName string     `json:"myEditedByName,omitempty"`
	// MyEditors: the teachers editing the student's work right now.
	MyEditors []models.ProjectEditor `json:"myEditors,omitempty"`
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
	// SubmissionNote is the student's text to the teacher; only for handed-in work, like Files.
	SubmissionNote string     `json:"submissionNote,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	SubmittedAt    *time.Time `json:"submittedAt,omitempty"`
	PreviewURL     string     `json:"previewUrl,omitempty"`
	// Late: handed in after the assignment's due date.
	Late bool `json:"late,omitempty"`
	// ReviewedBy, its name and ReviewedAt: who gave the current review, empty for work
	// reviewed before that was recorded.
	ReviewedBy     string     `json:"reviewedBy,omitempty"`
	ReviewedByName string     `json:"reviewedByName,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	// EditedBy, its name and EditedAt: the teacher who last changed the blocks, until the next
	// hand-in. CanEdit: the viewer may save the work now. CanRestore: they may also put the
	// student's version back.
	EditedBy     string     `json:"editedBy,omitempty"`
	EditedByName string     `json:"editedByName,omitempty"`
	EditedAt     *time.Time `json:"editedAt,omitempty"`
	CanEdit      bool       `json:"canEdit,omitempty"`
	CanRestore   bool       `json:"canRestore,omitempty"`
	// Editors: who has the work open in the editor right now.
	Editors []models.ProjectEditor `json:"editors,omitempty"`
}

// HandIn is what a hand-in adds to a review update.
type HandIn struct {
	At   time.Time
	Note string
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

// HistoryEventDTO is one line of the class history. Detail depends on Action, see
// models.ClassHistoryEventDB. A name is empty when the LMS no longer knows the user.
type HistoryEventDTO struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	ActorID    string    `json:"actorId"`
	ActorName  string    `json:"actorName,omitempty"`
	ActorRole  string    `json:"actorRole"`
	TargetID   string    `json:"targetId,omitempty"`
	TargetName string    `json:"targetName,omitempty"`
	Title      string    `json:"title,omitempty"`
	Detail     string    `json:"detail,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// HistoryPageDTO: NextCursor is empty on the last page.
type HistoryPageDTO struct {
	Items      []HistoryEventDTO `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
}

// HistoryCursor is the last event of a page: the next one starts after it.
type HistoryCursor struct {
	At time.Time
	ID string
}

type Gateway interface {
	CreateInvite(inv *models.ClassInviteDB) error
	UpdateInvite(inv *models.ClassInviteDB) error
	GetInviteByID(id string) (*models.ClassInviteDB, error)
	GetInviteBySlug(slug string) (*models.ClassInviteDB, error)
	GetInviteByCode(code string) (*models.ClassInviteDB, error)
	GetInviteByCourseCohort(courseID, cohortID string) (*models.ClassInviteDB, error)
	// ListInvitesByTeacher: the classes the teacher owns or co-teaches, newest first.
	ListInvitesByTeacher(teacherID string) ([]models.ClassInviteDB, error)
	ListAllInvites() ([]models.ClassInviteDB, error)
	ListActiveInvites() ([]models.ClassInviteDB, error)

	// Roster of classes not linked to the LMS.
	AddMember(classID, userID string) error
	RemoveMember(classID, userID string) error
	ListMembers(classID string) ([]models.ClassMemberDB, error)
	ListMemberClassIDs(userID string) ([]string, error)
	IsMemberOfAny(userID string, classIDs []string) (bool, error)

	// AddCoTeacher starts a term unless the teacher has a running one (ErrAlreadyCoTeacher) or
	// the class has maxActive (ErrTooManyCoTeachers).
	AddCoTeacher(row *models.ClassCoTeacherDB, maxActive int) error
	// IsCoTeacher: the teacher has a running term in the class.
	IsCoTeacher(classID, teacherID string) (bool, error)
	// WasCoTeacher: the teacher has any term in the class, over or not.
	WasCoTeacher(classID, teacherID string) (bool, error)
	// ListCoTeachers returns the running terms of the class, oldest first.
	ListCoTeachers(classID string) ([]models.ClassCoTeacherDB, error)
	// ListCoTeachings returns the teacher's running terms in classes that are not archived.
	ListCoTeachings(teacherID string) ([]models.ClassCoTeacherDB, error)
	// SetCoTeacherExpiry moves the end of a running term; remindedAt replaces the reminder mark.
	SetCoTeacherExpiry(classID, teacherID string, expiresAt time.Time, remindedAt *time.Time) (*models.ClassCoTeacherDB, error)
	RevokeCoTeacher(classID, teacherID, revokedBy string) error
	// ClaimExpiringCoTeachers marks the running terms ending by `before` as reminded and
	// returns them, so each is reminded of once however many backends run.
	ClaimExpiringCoTeachers(before time.Time) ([]models.ClassCoTeacherDB, error)

	AddHistoryEvent(e *models.ClassHistoryEventDB) error
	// ListHistoryEvents returns the class's events newest first, older than before when it is
	// set; no actions means all of them.
	ListHistoryEvents(classID string, actions []string, before *HistoryCursor, limit int) ([]models.ClassHistoryEventDB, error)

	CreateAssignment(a *models.ClassAssignmentDB) error
	ListAssignments(courseID, cohortID string) ([]models.ClassAssignmentDB, error)
	GetAssignment(id string) (*models.ClassAssignmentDB, error)
	SetAssignmentClosed(id string, closed bool) error
	// DeleteAssignment removes the assignment and its reviews; the students keep their work.
	DeleteAssignment(id string) error

	GetProject(id string) (*models.ScratchProjectDB, error)
	// ForkProjectForStudent with a nil template issues an empty project.
	ForkProjectForStudent(template *models.ScratchProjectDB, studentUserID, title string, assignment *models.ClassAssignmentDB) (*models.ScratchProjectDB, error)
	// HasOwnSave reports whether the project has a version other than the one it was issued with.
	HasOwnSave(projectID string) (bool, error)
	ListSubmissions(assignmentID string) ([]models.ScratchProjectDB, error)
	// ListActiveEditors returns who is editing these projects now (models.EditingTTLSeconds).
	ListActiveEditors(projectIDs []string) ([]models.ScratchProjectEditorDB, error)
	ListSubmissionsForAssignments(assignmentIDs []string) ([]models.ScratchProjectDB, error)
	ListProjectsByCohortAssignment(cohortID, assignmentID string) ([]models.ScratchProjectDB, error)
	// UpdateReview with a nil handIn is reviewerID's review: it keeps the time and the note of
	// the last hand-in. A hand-in clears the reviewer.
	UpdateReview(projectID, status, comment, reviewerID string, handIn *HandIn) error
	// SetProjectNote replaces the student's note of work on review (ErrNotOnReview otherwise).
	// The scratch_projects trigger moves updated_at, as it does on a review.
	SetProjectNote(projectID, note string) error
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
	UpdateFileReview(submissionID, status, comment, reviewerID string, handIn *HandIn) error
	// SetFileSubmissionNote is SetProjectNote for a file submission; updated_at stays.
	SetFileSubmissionNote(submissionID, note string) error
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

	// Co-teachers: the owner or a SuperAdmin appoints, extends and removes; a co-teacher may
	// list them and remove themselves.
	ListCoTeachers(teacherID string, role models.Role, classID string) ([]CoTeacherDTO, error)
	AddCoTeacher(teacherID string, role models.Role, classID, email string, expiresAt time.Time) (*CoTeacherDTO, error)
	ExtendCoTeacher(teacherID string, role models.Role, classID, coTeacherID string, expiresAt time.Time) (*CoTeacherDTO, error)
	RemoveCoTeacher(teacherID string, role models.Role, classID, coTeacherID string) error
	// RemindExpiringCoTeachers tells both sides that a term ends within a day; returns how many.
	RemindExpiringCoTeachers(now time.Time) (int, error)

	// ListHistory is a page of the class history for its teachers, see historyCategories.
	ListHistory(teacherID string, role models.Role, classID, category, cursor string) (*HistoryPageDTO, error)
	// RecordClassMessage puts a message sent from the class board into the history.
	RecordClassMessage(teacherID string, role models.Role, classID string, recipientIDs []string)
	// RecordTeacherEdit tells the class about a teacher's change to handed-in Scratch work.
	RecordTeacherEdit(teacherID string, p *models.ScratchProjectDB, restored bool)

	ListMembers(teacherID string, role models.Role, classID string) ([]CohortUserDTO, error)
	AddMembersByEmail(teacherID string, role models.Role, classID string, emails []string) (added []string, missing []string, err error)
	RemoveMember(teacherID string, role models.Role, classID, username string) error

	CreateAssignment(teacherID string, role models.Role, classID string, in CreateAssignmentInput) (*AssignmentDTO, error)
	ListAssignments(viewerID string, role models.Role, classID string) ([]AssignmentDTO, error)
	IssueToLatecomers(teacherID string, role models.Role, classID, assignmentID string) (issued int, err error)
	SetAssignmentClosed(teacherID string, role models.Role, classID, assignmentID string, closed bool) (*AssignmentDTO, error)
	// DeleteAssignment cannot be undone: the reviews go, the students' projects and files stay.
	DeleteAssignment(teacherID string, role models.Role, classID, assignmentID string) error
	// SubmitAssignment and SubmitFiles hand the work in; note is an optional text to the teacher.
	SubmitAssignment(studentID, projectID, note string) error
	// UpdateProjectNote and UpdateFileNote change that text while the work waits for review.
	UpdateProjectNote(studentID, projectID, note string) error
	ReviewSubmission(teacherID string, role models.Role, projectID, status, comment string) error
	ListSubmissions(teacherID string, role models.Role, classID, assignmentID string) ([]SubmissionDTO, error)
	ProgressMatrix(teacherID string, role models.Role, classID string) (assignments []AssignmentDTO, rows []ProgressRowDTO, err error)

	// File assignments.
	FileAssignmentsEnabled() bool
	AttachFile(studentID, assignmentID, fileID string) (*AssignmentDTO, error)
	DetachFile(studentID, assignmentID, fileID string) (*AssignmentDTO, error)
	SubmitFiles(studentID, assignmentID, note string) error
	UpdateFileNote(studentID, assignmentID, note string) error
	// OpenStarterFile is the assignment's starter file for the class's teachers or a member.
	OpenStarterFile(viewerID string, role models.Role, assignmentID string) (*models.UserFileDB, io.ReadSeekCloser, error)
	// OpenSubmissionFile is a file of handed-in work for the class's teachers (any work for a SuperAdmin).
	OpenSubmissionFile(teacherID string, role models.Role, submissionID, fileID string) (*models.UserFileDB, io.ReadSeekCloser, error)
	ReviewFileSubmission(teacherID string, role models.Role, submissionID, status, comment string) error
}
