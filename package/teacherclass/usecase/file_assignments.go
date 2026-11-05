package usecase

import (
	"errors"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const defaultMaxSubmissionFiles = 10

// maxSubmissionFiles is how many files one student may attach to a file assignment.
func maxSubmissionFiles() int {
	if n := viper.GetInt("classes.maxSubmissionFiles"); n > 0 {
		return n
	}
	return defaultMaxSubmissionFiles
}

// FileAssignmentsEnabled: files may go from a student to the teacher and from the teacher to
// the class, which "Мои файлы" allows only behind a required antivirus scan.
func (u *UseCaseImpl) FileAssignmentsEnabled() bool {
	return u.files != nil && u.files.ExchangeEnabled()
}

// fileErr turns a "Мои файлы" error into one of this package, keeping the scan and type
// errors the ЛК shows as they are.
func fileErr(err error) error {
	switch {
	case errors.Is(err, userfiles.ErrFileNotFound):
		return teacherclass.ErrNotFound
	case errors.Is(err, userfiles.ErrDisabled):
		return teacherclass.ErrFileAssignmentsDisabled
	}
	return err
}

// normalizeKinds checks the kinds a teacher allows against those "Мои файлы" stores.
func (u *UseCaseImpl) normalizeKinds(kinds []string) (pq.StringArray, error) {
	accepted := u.files.AcceptedKinds()
	out := pq.StringArray{}
	for _, k := range kinds {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || slices.Contains(out, k) {
			continue
		}
		if !slices.Contains(accepted, k) {
			return nil, teacherclass.ErrBadRequest
		}
		out = append(out, k)
	}
	slices.Sort(out)
	return out, nil
}

// prepareFileAssignment validates a new file assignment and fills its file-only fields.
func (u *UseCaseImpl) prepareFileAssignment(teacherID string, in teacherclass.CreateAssignmentInput, a *models.ClassAssignmentDB) (*models.UserFileDB, error) {
	if !u.FileAssignmentsEnabled() {
		return nil, teacherclass.ErrFileAssignmentsDisabled
	}
	if strings.TrimSpace(in.TemplateProjectID) != "" {
		return nil, teacherclass.ErrBadRequest
	}
	kinds, err := u.normalizeKinds(in.AllowedKinds)
	if err != nil {
		return nil, err
	}
	a.Kind = models.AssignmentFile
	a.AllowedKinds = kinds
	id := strings.TrimSpace(in.TemplateFileID)
	if id == "" {
		return nil, nil
	}
	// The class downloads the starter file: it is scanned first, like a handed-in file.
	f, err := u.files.EnsureScanned(teacherID, id)
	if err != nil {
		return nil, fileErr(err)
	}
	a.TemplateFileID = &f.ID
	return f, nil
}

// fillTemplateFiles adds what a class sees of a file assignment: its starter file.
func (u *UseCaseImpl) fillTemplateFiles(dtos []teacherclass.AssignmentDTO, list []models.ClassAssignmentDB) error {
	var ids []string
	for i := range list {
		if list[i].IsFileAssignment() && list[i].TemplateFileID != nil {
			ids = append(ids, *list[i].TemplateFileID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	files, err := u.gw.GetUserFiles(ids)
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].TemplateFileID == nil {
			continue
		}
		// A starter file its owner deleted (moderation can) is simply gone.
		if f, ok := files[*list[i].TemplateFileID]; ok {
			ref := teacherclass.FileRef(&f)
			dtos[i].TemplateFile = &ref
		}
	}
	return nil
}

func fileRefs(files []models.UserFileDB) []teacherclass.FileRefDTO {
	out := make([]teacherclass.FileRefDTO, 0, len(files))
	for i := range files {
		out = append(out, teacherclass.FileRef(&files[i]))
	}
	return out
}

// fillMyFileWork adds the student's submission to the student view of a file assignment.
func (u *UseCaseImpl) fillMyFileWork(dto *teacherclass.AssignmentDTO, assignmentID, studentID string) error {
	sub, err := u.gw.FindFileSubmission(assignmentID, studentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	files, err := u.gw.ListSubmissionFiles([]string{sub.ID})
	if err != nil {
		return err
	}
	dto.MySubmissionID = sub.ID
	dto.MyReviewStatus = sub.ReviewStatus
	dto.MyReviewComment = sub.ReviewComment
	dto.MySubmissionNote = sub.SubmissionNote
	dto.MyFiles = fileRefs(files[sub.ID])
	return nil
}

// isClassMember: the user is on the class roster (the Open edX cohort for a linked class).
func (u *UseCaseImpl) isClassMember(userID string, inv *models.ClassInviteDB) (bool, error) {
	if inv.LmsLinked {
		return studentInCohort(userID, inv.EdxCohortID)
	}
	return u.gw.IsMemberOfAny(userID, []string{inv.ID})
}

// studentFileAssignment is a file assignment of a class the student is in.
func (u *UseCaseImpl) studentFileAssignment(studentID, assignmentID string) (*models.ClassAssignmentDB, error) {
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	if !a.IsFileAssignment() {
		return nil, teacherclass.ErrBadRequest
	}
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	member, err := u.isClassMember(studentID, inv)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, teacherclass.ErrForbidden
	}
	if !u.FileAssignmentsEnabled() {
		return nil, teacherclass.ErrFileAssignmentsDisabled
	}
	return a, nil
}

// editableSubmission: the student may change the files of a draft or of returned work.
func editableSubmission(sub *models.ClassFileSubmissionDB) error {
	switch sub.ReviewStatus {
	case "", models.ReviewDraft, models.ReviewReturned:
		return nil
	}
	return teacherclass.ErrAlreadyHandedIn
}

func (u *UseCaseImpl) studentAssignmentDTO(a *models.ClassAssignmentDB, studentID string) (*teacherclass.AssignmentDTO, error) {
	dto := assignmentDTO(a)
	if err := u.fillTemplateFiles([]teacherclass.AssignmentDTO{dto}, []models.ClassAssignmentDB{*a}); err != nil {
		return nil, err
	}
	if err := u.fillMyFileWork(&dto, a.ID, studentID); err != nil {
		return nil, err
	}
	return &dto, nil
}

func (u *UseCaseImpl) AttachFile(studentID, assignmentID, fileID string) (*teacherclass.AssignmentDTO, error) {
	a, err := u.studentFileAssignment(studentID, assignmentID)
	if err != nil {
		return nil, err
	}
	if a.ClosedAt != nil {
		return nil, teacherclass.ErrAssignmentClosed
	}
	if sub, err := u.gw.FindFileSubmission(a.ID, studentID); err == nil {
		if err := editableSubmission(sub); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// Only the student's own file; one uploaded before the scanner is scanned now.
	f, err := u.files.EnsureScanned(studentID, fileID)
	if err != nil {
		return nil, fileErr(err)
	}
	if len(a.AllowedKinds) > 0 && !slices.Contains([]string(a.AllowedKinds), f.Kind) {
		return nil, teacherclass.ErrFileKindNotAllowed
	}
	sub, err := u.gw.EnsureFileSubmission(a.ID, studentID)
	if err != nil {
		return nil, err
	}
	if err := u.gw.AttachSubmissionFile(sub.ID, f.ID, maxSubmissionFiles()); err != nil {
		return nil, err
	}
	return u.studentAssignmentDTO(a, studentID)
}

func (u *UseCaseImpl) DetachFile(studentID, assignmentID, fileID string) (*teacherclass.AssignmentDTO, error) {
	a, err := u.studentFileAssignment(studentID, assignmentID)
	if err != nil {
		return nil, err
	}
	if a.ClosedAt != nil {
		return nil, teacherclass.ErrAssignmentClosed
	}
	sub, err := u.gw.FindFileSubmission(a.ID, studentID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	if err := editableSubmission(sub); err != nil {
		return nil, err
	}
	if err := u.gw.DetachSubmissionFile(sub.ID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	return u.studentAssignmentDTO(a, studentID)
}

func (u *UseCaseImpl) UpdateFileNote(studentID, assignmentID, note string) error {
	note, err := submissionNote(note)
	if err != nil {
		return err
	}
	a, err := u.studentFileAssignment(studentID, assignmentID)
	if err != nil {
		return err
	}
	if a.ClosedAt != nil {
		return teacherclass.ErrAssignmentClosed
	}
	sub, err := u.gw.FindFileSubmission(a.ID, studentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return teacherclass.ErrNotOnReview
	}
	if err != nil {
		return err
	}
	return u.gw.SetFileSubmissionNote(sub.ID, note)
}

func (u *UseCaseImpl) SubmitFiles(studentID, assignmentID, note string) error {
	note, err := submissionNote(note)
	if err != nil {
		return err
	}
	a, err := u.studentFileAssignment(studentID, assignmentID)
	if err != nil {
		return err
	}
	sub, err := u.gw.FindFileSubmission(a.ID, studentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return teacherclass.ErrNoSubmissionFiles
	}
	if err != nil {
		return err
	}
	if err := editableSubmission(sub); err != nil {
		return err
	}
	// The due date only marks late work; closing the assignment stops hand-ins.
	if a.ClosedAt != nil {
		return teacherclass.ErrAssignmentClosed
	}
	files, err := u.gw.ListSubmissionFiles([]string{sub.ID})
	if err != nil {
		return err
	}
	if len(files[sub.ID]) == 0 {
		return teacherclass.ErrNoSubmissionFiles
	}
	handIn := &teacherclass.HandIn{At: time.Now().UTC(), Note: note}
	if err := u.gw.UpdateFileReview(sub.ID, models.ReviewSubmitted, "", "", handIn); err != nil {
		return err
	}
	u.notifyWorkSubmitted(studentID, a)
	return nil
}

func (u *UseCaseImpl) OpenStarterFile(viewerID string, role models.Role, assignmentID string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil || !a.IsFileAssignment() || a.TemplateFileID == nil {
		return nil, nil, teacherclass.ErrNotFound
	}
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		return nil, nil, teacherclass.ErrNotFound
	}
	teaches, err := u.canTeach(inv, viewerID, role)
	if err != nil {
		return nil, nil, err
	}
	if !teaches {
		member, err := u.isClassMember(viewerID, inv)
		if err != nil {
			return nil, nil, err
		}
		if !member {
			return nil, nil, teacherclass.ErrForbidden
		}
	}
	if !u.FileAssignmentsEnabled() {
		return nil, nil, teacherclass.ErrFileAssignmentsDisabled
	}
	row, rc, err := u.files.Open(a.CreatedByTeacherID, *a.TemplateFileID)
	if err != nil {
		return nil, nil, fileErr(err)
	}
	return row, rc, nil
}

// managedFileSubmission is a submission of a class the teacher manages, with its assignment.
func (u *UseCaseImpl) managedFileSubmission(teacherID string, role models.Role, submissionID string) (*models.ClassFileSubmissionDB, *models.ClassAssignmentDB, error) {
	sub, err := u.gw.GetFileSubmission(submissionID)
	if err != nil {
		return nil, nil, teacherclass.ErrNotFound
	}
	a, err := u.gw.GetAssignment(sub.AssignmentID)
	if err != nil {
		return nil, nil, teacherclass.ErrNotFound
	}
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		return nil, nil, teacherclass.ErrForbidden
	}
	if teaches, err := u.canTeach(inv, teacherID, role); err != nil {
		return nil, nil, err
	} else if !teaches {
		return nil, nil, teacherclass.ErrForbidden
	}
	// A draft is the student's own work, as with Scratch copies.
	if !draftVisibleStatus(role, sub.ReviewStatus) {
		return nil, nil, teacherclass.ErrNotFound
	}
	return sub, a, nil
}

func (u *UseCaseImpl) OpenSubmissionFile(teacherID string, role models.Role, submissionID, fileID string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	sub, _, err := u.managedFileSubmission(teacherID, role, submissionID)
	if err != nil {
		return nil, nil, err
	}
	if !u.FileAssignmentsEnabled() {
		return nil, nil, teacherclass.ErrFileAssignmentsDisabled
	}
	files, err := u.gw.ListSubmissionFiles([]string{sub.ID})
	if err != nil {
		return nil, nil, err
	}
	if !slices.ContainsFunc(files[sub.ID], func(f models.UserFileDB) bool { return f.ID == fileID }) {
		return nil, nil, teacherclass.ErrNotFound
	}
	row, rc, err := u.files.Open(sub.StudentUserID, fileID)
	if err != nil {
		return nil, nil, fileErr(err)
	}
	return row, rc, nil
}

func (u *UseCaseImpl) ReviewFileSubmission(teacherID string, role models.Role, submissionID, status, comment string) error {
	if !reviewStatusAllowed(status) {
		return teacherclass.ErrBadRequest
	}
	sub, a, err := u.managedFileSubmission(teacherID, role, submissionID)
	if err != nil {
		return err
	}
	if err := u.gw.UpdateFileReview(sub.ID, status, comment, teacherID, nil); err != nil {
		return err
	}
	u.notifyWorkReviewed(sub.StudentUserID, sub.ID, a.Title, status, comment)
	return nil
}

// fileSubmissionDTOs is ListSubmissions for a file assignment.
func (u *UseCaseImpl) fileSubmissionDTOs(role models.Role, a *models.ClassAssignmentDB) ([]teacherclass.SubmissionDTO, error) {
	subs, err := u.gw.ListFileSubmissions([]string{a.ID})
	if err != nil {
		return nil, err
	}
	var visible []string
	for i := range subs {
		if draftVisibleStatus(role, subs[i].ReviewStatus) {
			visible = append(visible, subs[i].ID)
		}
	}
	files, err := u.gw.ListSubmissionFiles(visible)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.SubmissionDTO, 0, len(subs))
	for i := range subs {
		s := &subs[i]
		dto := teacherclass.SubmissionDTO{
			SubmissionID: s.ID,
			OwnerUserID:  s.StudentUserID,
			Title:        a.Title,
			ReviewStatus: s.ReviewStatus,
			UpdatedAt:    s.UpdatedAt,
		}
		// As with Scratch drafts: the board shows the status and the last change, not the files.
		if draftVisibleStatus(role, s.ReviewStatus) {
			dto.ReviewComment = s.ReviewComment
			dto.SubmissionNote = s.SubmissionNote
			dto.SubmittedAt = s.SubmittedAt
			dto.Late = lateAt(a, s.SubmittedAt)
			dto.Files = fileRefs(files[s.ID])
			dto.ReviewedBy = deref(s.ReviewedByTeacherID)
			dto.ReviewedAt = s.ReviewedAt
		}
		out = append(out, dto)
	}
	fillReviewerNames(out)
	return out, nil
}

// fileWork is ProgressMatrix input for file assignments: one entry per submission.
func (u *UseCaseImpl) fileWork(role models.Role, assignments []models.ClassAssignmentDB) (map[string][]work, error) {
	var ids []string
	byID := map[string]*models.ClassAssignmentDB{}
	for i := range assignments {
		if assignments[i].IsFileAssignment() {
			ids = append(ids, assignments[i].ID)
			byID[assignments[i].ID] = &assignments[i]
		}
	}
	out := map[string][]work{}
	if len(ids) == 0 {
		return out, nil
	}
	subs, err := u.gw.ListFileSubmissions(ids)
	if err != nil {
		return nil, err
	}
	for i := range subs {
		s := &subs[i]
		w := work{owner: s.StudentUserID, status: s.ReviewStatus}
		if draftVisibleStatus(role, s.ReviewStatus) {
			w.submissionID = s.ID
			w.late = lateAt(byID[s.AssignmentID], s.SubmittedAt)
		}
		out[s.AssignmentID] = append(out[s.AssignmentID], w)
	}
	return out, nil
}
