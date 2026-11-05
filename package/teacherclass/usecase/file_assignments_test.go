package usecase

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"gorm.io/gorm"
)

// fileGateway is one ЛК class "class-1" of teacher "10" with student "797" on the roster.
type fileGateway struct {
	teacherclass.Gateway
	inv         *models.ClassInviteDB
	members     map[string]bool
	assignments map[string]*models.ClassAssignmentDB
	subs        map[string]*models.ClassFileSubmissionDB
	links       map[string][]string // submission -> file ids
	files       map[string]models.UserFileDB
	maxSeen     int
}

func newFileGateway() *fileGateway {
	return &fileGateway{
		inv: &models.ClassInviteDB{
			ID: "class-1", EdxCohortID: models.UnlinkedCohortKey("class-1"), OwnerTeacherID: "10",
		},
		members:     map[string]bool{"797": true},
		assignments: map[string]*models.ClassAssignmentDB{},
		subs:        map[string]*models.ClassFileSubmissionDB{},
		links:       map[string][]string{},
		files:       map[string]models.UserFileDB{},
	}
}

func (g *fileGateway) GetInviteByID(id string) (*models.ClassInviteDB, error) {
	if id == g.inv.ID {
		return g.inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *fileGateway) GetInviteByCourseCohort(courseID, cohortID string) (*models.ClassInviteDB, error) {
	if courseID == g.inv.EdxCourseID && cohortID == g.inv.EdxCohortID {
		return g.inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *fileGateway) IsMemberOfAny(userID string, classIDs []string) (bool, error) {
	return g.members[userID] && len(classIDs) == 1 && classIDs[0] == g.inv.ID, nil
}

func (g *fileGateway) CreateAssignment(a *models.ClassAssignmentDB) error {
	a.ID = "a" + string(rune('0'+len(g.assignments)+1))
	a.CreatedAt = time.Now()
	g.assignments[a.ID] = a
	return nil
}

func (g *fileGateway) GetAssignment(id string) (*models.ClassAssignmentDB, error) {
	if a, ok := g.assignments[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *fileGateway) ListAssignments(courseID, cohortID string) ([]models.ClassAssignmentDB, error) {
	var out []models.ClassAssignmentDB
	for _, a := range g.assignments {
		out = append(out, *a)
	}
	return out, nil
}

func (g *fileGateway) ListSubmissionsForAssignments([]string) ([]models.ScratchProjectDB, error) {
	return nil, nil
}

func (g *fileGateway) GetFileSubmission(id string) (*models.ClassFileSubmissionDB, error) {
	if s, ok := g.subs[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *fileGateway) FindFileSubmission(assignmentID, studentID string) (*models.ClassFileSubmissionDB, error) {
	for _, s := range g.subs {
		if s.AssignmentID == assignmentID && s.StudentUserID == studentID {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *fileGateway) EnsureFileSubmission(assignmentID, studentID string) (*models.ClassFileSubmissionDB, error) {
	if s, err := g.FindFileSubmission(assignmentID, studentID); err == nil {
		return s, nil
	}
	s := &models.ClassFileSubmissionDB{
		ID: "s-" + studentID, AssignmentID: assignmentID, StudentUserID: studentID, ReviewStatus: models.ReviewDraft,
	}
	g.subs[s.ID] = s
	return s, nil
}

func (g *fileGateway) ListFileSubmissions(ids []string) ([]models.ClassFileSubmissionDB, error) {
	var out []models.ClassFileSubmissionDB
	for _, s := range g.subs {
		for _, id := range ids {
			if s.AssignmentID == id {
				out = append(out, *s)
			}
		}
	}
	return out, nil
}

func (g *fileGateway) ListSubmissionFiles(ids []string) (map[string][]models.UserFileDB, error) {
	out := map[string][]models.UserFileDB{}
	for _, id := range ids {
		for _, fid := range g.links[id] {
			out[id] = append(out[id], g.files[fid])
		}
	}
	return out, nil
}

func (g *fileGateway) AttachSubmissionFile(subID, fileID string, maxFiles int) error {
	g.maxSeen = maxFiles
	for _, id := range g.links[subID] {
		if id == fileID {
			return nil
		}
	}
	if len(g.links[subID]) >= maxFiles {
		return teacherclass.ErrTooManySubmissionFiles
	}
	g.links[subID] = append(g.links[subID], fileID)
	return nil
}

func (g *fileGateway) DetachSubmissionFile(subID, fileID string) error {
	for i, id := range g.links[subID] {
		if id == fileID {
			g.links[subID] = append(g.links[subID][:i], g.links[subID][i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (g *fileGateway) UpdateFileReview(subID, status, comment string, handIn *teacherclass.HandIn) error {
	s := g.subs[subID]
	s.ReviewStatus, s.ReviewComment = status, comment
	if handIn != nil {
		s.SubmittedAt = &handIn.At
		s.SubmissionNote = handIn.Note
	}
	return nil
}

func (g *fileGateway) SetFileSubmissionNote(subID, note string) error {
	if s := g.subs[subID]; s.ReviewStatus == models.ReviewSubmitted {
		s.SubmissionNote = note
		return nil
	}
	return teacherclass.ErrNotOnReview
}

func (g *fileGateway) GetUserFiles(ids []string) (map[string]models.UserFileDB, error) {
	out := map[string]models.UserFileDB{}
	for _, id := range ids {
		if f, ok := g.files[id]; ok {
			out[id] = f
		}
	}
	return out, nil
}

// fakeFiles is "Мои файлы" with exchange on: files belong to the owner in their row.
type fakeFiles struct {
	userfiles.UseCase
	gw       *fileGateway
	exchange bool
	infected map[string]bool
}

func (f *fakeFiles) ExchangeEnabled() bool   { return f.exchange }
func (f *fakeFiles) AcceptedKinds() []string { return []string{"fcstd", "ino", "pdf", "stl"} }

func (f *fakeFiles) EnsureScanned(owner, id string) (*models.UserFileDB, error) {
	row, ok := f.gw.files[id]
	if !ok || row.OwnerUserID != owner {
		return nil, userfiles.ErrFileNotFound
	}
	if f.infected[id] {
		return nil, userfiles.ErrFileInfected
	}
	return &row, nil
}

type nopReadSeekCloser struct{ io.ReadSeeker }

func (nopReadSeekCloser) Close() error { return nil }

func (f *fakeFiles) Open(owner, id string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	row, ok := f.gw.files[id]
	if !ok || row.OwnerUserID != owner {
		return nil, nil, userfiles.ErrFileNotFound
	}
	return &row, nopReadSeekCloser{strings.NewReader("bytes")}, nil
}

func newFileUseCase(t *testing.T) (*UseCaseImpl, *fileGateway) {
	t.Helper()
	withoutLMSNames(t)
	gw := newFileGateway()
	gw.files["tpl"] = models.UserFileDB{ID: "tpl", OwnerUserID: "10", OriginalName: "base.FCStd", Kind: "fcstd"}
	gw.files["f1"] = models.UserFileDB{ID: "f1", OwnerUserID: "797", OriginalName: "part.FCStd", Kind: "fcstd"}
	gw.files["f2"] = models.UserFileDB{ID: "f2", OwnerUserID: "797", OriginalName: "part.stl", Kind: "stl"}
	gw.files["f3"] = models.UserFileDB{ID: "f3", OwnerUserID: "797", OriginalName: "notes.pdf", Kind: "pdf"}
	gw.files["other"] = models.UserFileDB{ID: "other", OwnerUserID: "555", OriginalName: "x.stl", Kind: "stl"}
	return &UseCaseImpl{gw: gw, files: &fakeFiles{gw: gw, exchange: true}, notify: &workNotifier{}}, gw
}

func createFileAssignment(t *testing.T, uc *UseCaseImpl) *teacherclass.AssignmentDTO {
	t.Helper()
	dto, err := uc.CreateAssignment("10", models.Teacher, "class-1", teacherclass.CreateAssignmentInput{
		Title: "Деталь", AssignmentKey: "detal", Kind: models.AssignmentFile,
		AllowedKinds: []string{"STL", "fcstd", "stl"}, TemplateFileID: "tpl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return dto
}

func TestCreateFileAssignment(t *testing.T) {
	uc, gw := newFileUseCase(t)
	dto := createFileAssignment(t, uc)
	if dto.Kind != models.AssignmentFile || strings.Join(dto.AllowedKinds, ",") != "fcstd,stl" {
		t.Fatalf("dto = %+v", dto)
	}
	if dto.TemplateFile == nil || dto.TemplateFile.Name != "base.FCStd" || dto.CopiesIssued != 0 {
		t.Fatalf("starter file / copies: %+v", dto)
	}
	if a := gw.assignments[dto.ID]; a.TemplateFileID == nil || *a.TemplateFileID != "tpl" {
		t.Fatalf("stored assignment = %+v", a)
	}
}

func TestCreateFileAssignmentRefused(t *testing.T) {
	uc, _ := newFileUseCase(t)
	base := teacherclass.CreateAssignmentInput{Title: "T", AssignmentKey: "t", Kind: models.AssignmentFile}

	in := base
	in.AllowedKinds = []string{"exe"}
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", in); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("unknown kind: %v", err)
	}
	in = base
	in.TemplateFileID = "f1" // the student's file, not the teacher's
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", in); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("someone else's starter file: %v", err)
	}
	in = base
	in.TemplateProjectID = "p1"
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", in); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("Scratch template on a file assignment: %v", err)
	}
	in = base
	in.Kind = "video"
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", in); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("unknown assignment kind: %v", err)
	}

	uc.files.(*fakeFiles).exchange = false
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", base); !errors.Is(err, teacherclass.ErrFileAssignmentsDisabled) {
		t.Fatalf("without the scanner: %v", err)
	}
	uc.files = nil
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", base); !errors.Is(err, teacherclass.ErrFileAssignmentsDisabled) {
		t.Fatalf("without user files: %v", err)
	}
}

func TestAttachAndSubmitFiles(t *testing.T) {
	uc, gw := newFileUseCase(t)
	a := createFileAssignment(t, uc)

	if err := uc.UpdateFileNote("797", a.ID, "x"); !errors.Is(err, teacherclass.ErrNotOnReview) {
		t.Fatalf("a note before any work: %v", err)
	}
	if err := uc.SubmitFiles("797", a.ID, ""); !errors.Is(err, teacherclass.ErrNoSubmissionFiles) {
		t.Fatalf("submit before any file: %v", err)
	}
	if _, err := uc.AttachFile("797", a.ID, "f3"); !errors.Is(err, teacherclass.ErrFileKindNotAllowed) {
		t.Fatalf("pdf on an fcstd/stl assignment: %v", err)
	}
	if _, err := uc.AttachFile("797", a.ID, "other"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("someone else's file: %v", err)
	}
	if _, err := uc.AttachFile("555", a.ID, "other"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("not in the class: %v", err)
	}
	uc.files.(*fakeFiles).infected = map[string]bool{"f2": true}
	if _, err := uc.AttachFile("797", a.ID, "f2"); !errors.Is(err, userfiles.ErrFileInfected) {
		t.Fatalf("infected file: %v", err)
	}
	uc.files.(*fakeFiles).infected = nil

	dto, err := uc.AttachFile("797", a.ID, "f1")
	if err != nil {
		t.Fatal(err)
	}
	if dto, err = uc.AttachFile("797", a.ID, "f2"); err != nil {
		t.Fatal(err)
	}
	if len(dto.MyFiles) != 2 || dto.MyReviewStatus != models.ReviewDraft || dto.MySubmissionID == "" {
		t.Fatalf("student view after attach: %+v", dto)
	}
	if gw.maxSeen != defaultMaxSubmissionFiles {
		t.Fatalf("attach limit = %d, want the default %d", gw.maxSeen, defaultMaxSubmissionFiles)
	}

	// The teacher sees a draft without its files and may not download them.
	subs, err := uc.ListSubmissions("10", models.Teacher, "class-1", a.ID)
	if err != nil || len(subs) != 1 || subs[0].Files != nil || subs[0].ReviewStatus != models.ReviewDraft {
		t.Fatalf("draft on the board: %+v, %v", subs, err)
	}
	if _, _, err := uc.OpenSubmissionFile("10", models.Teacher, dto.MySubmissionID, "f1"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("teacher opening a draft: %v", err)
	}

	if err := uc.SubmitFiles("797", a.ID, strings.Repeat("я", maxSubmissionNoteChars+1)); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("a note over the limit: %v", err)
	}
	if err := uc.SubmitFiles("797", a.ID, " Крепление в part.stl "); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AttachFile("797", a.ID, "f1"); !errors.Is(err, teacherclass.ErrAlreadyHandedIn) {
		t.Fatalf("attach after hand-in: %v", err)
	}
	if _, err := uc.DetachFile("797", a.ID, "f1"); !errors.Is(err, teacherclass.ErrAlreadyHandedIn) {
		t.Fatalf("detach after hand-in: %v", err)
	}

	subs, err = uc.ListSubmissions("10", models.Teacher, "class-1", a.ID)
	if err != nil || len(subs[0].Files) != 2 || subs[0].SubmittedAt == nil || subs[0].SubmissionNote != "Крепление в part.stl" {
		t.Fatalf("handed-in work on the board: %+v, %v", subs, err)
	}
	// While the work waits for review the student may change the note, and only that.
	if err := uc.UpdateFileNote("797", a.ID, " Крепление в part.stl "); err != nil {
		t.Fatal(err)
	}
	// A review keeps the note: it is what the student said about this hand-in.
	if err := uc.ReviewFileSubmission("10", models.Teacher, dto.MySubmissionID, models.ReviewSubmitted, "смотрю"); err != nil {
		t.Fatal(err)
	}
	mine, err := uc.ListAssignments("797", models.Student, "class-1")
	if err != nil || len(mine) != 1 || mine[0].MySubmissionNote != "Крепление в part.stl" {
		t.Fatalf("the student's view of the note: %+v, %v", mine, err)
	}
	row, rc, err := uc.OpenSubmissionFile("10", models.Teacher, dto.MySubmissionID, "f2")
	if err != nil || row.ID != "f2" {
		t.Fatalf("teacher download: %+v, %v", row, err)
	}
	rc.Close()
	if _, _, err := uc.OpenSubmissionFile("10", models.Teacher, dto.MySubmissionID, "f3"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("a file not in the submission: %v", err)
	}
	if _, _, err := uc.OpenSubmissionFile("99", models.Teacher, dto.MySubmissionID, "f2"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("another teacher: %v", err)
	}

	// Returned for rework: the student changes the files and hands in again.
	if err := uc.ReviewFileSubmission("10", models.Teacher, dto.MySubmissionID, models.ReviewReturned, "Добавь крепление"); err != nil {
		t.Fatal(err)
	}
	if err := uc.ReviewFileSubmission("10", models.Teacher, dto.MySubmissionID, models.ReviewDraft, ""); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("teacher setting draft: %v", err)
	}
	if dto, err = uc.DetachFile("797", a.ID, "f2"); err != nil || len(dto.MyFiles) != 1 {
		t.Fatalf("detach returned work: %+v, %v", dto, err)
	}
	if dto.MyReviewComment != "Добавь крепление" {
		t.Fatalf("comment = %q", dto.MyReviewComment)
	}
	if err := uc.SubmitFiles("797", a.ID, ""); err != nil {
		t.Fatal(err)
	}
}

func TestFileAssignmentClosed(t *testing.T) {
	uc, gw := newFileUseCase(t)
	a := createFileAssignment(t, uc)
	if _, err := uc.AttachFile("797", a.ID, "f1"); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	gw.assignments[a.ID].ClosedAt = &now
	if _, err := uc.AttachFile("797", a.ID, "f2"); !errors.Is(err, teacherclass.ErrAssignmentClosed) {
		t.Fatalf("attach to a closed assignment: %v", err)
	}
	if err := uc.SubmitFiles("797", a.ID, ""); !errors.Is(err, teacherclass.ErrAssignmentClosed) {
		t.Fatalf("submit to a closed assignment: %v", err)
	}
	if _, err := uc.IssueToLatecomers("10", models.Teacher, "class-1", a.ID); err == nil {
		t.Fatal("issue-late on a file assignment should fail")
	}
}

func TestStarterFileAccess(t *testing.T) {
	uc, _ := newFileUseCase(t)
	a := createFileAssignment(t, uc)
	if row, rc, err := uc.OpenStarterFile("797", models.Student, a.ID); err != nil || row.ID != "tpl" {
		t.Fatalf("member: %+v, %v", row, err)
	} else {
		rc.Close()
	}
	if _, rc, err := uc.OpenStarterFile("10", models.Teacher, a.ID); err != nil {
		t.Fatalf("owner: %v", err)
	} else {
		rc.Close()
	}
	if _, _, err := uc.OpenStarterFile("555", models.Student, a.ID); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("not in the class: %v", err)
	}
}

func TestStudentListAndProgressWithFiles(t *testing.T) {
	uc, _ := newFileUseCase(t)
	a := createFileAssignment(t, uc)
	if _, err := uc.AttachFile("797", a.ID, "f1"); err != nil {
		t.Fatal(err)
	}
	list, err := uc.ListAssignments("797", models.Student, "class-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("student list: %+v, %v", list, err)
	}
	if list[0].TemplateFile == nil || len(list[0].MyFiles) != 1 || list[0].MaxFiles != defaultMaxSubmissionFiles {
		t.Fatalf("student view: %+v", list[0])
	}

	_, rows, err := uc.ProgressMatrix("10", models.Teacher, "class-1")
	if err != nil || len(rows) != 1 || rows[0].UserID != "797" {
		t.Fatalf("progress: %+v, %v", rows, err)
	}
	if c := rows[0].Cells[0]; c.Status != models.ReviewDraft || c.SubmissionID != "" {
		t.Fatalf("draft cell = %+v, want no link", c)
	}
	if err := uc.SubmitFiles("797", a.ID, ""); err != nil {
		t.Fatal(err)
	}
	_, rows, _ = uc.ProgressMatrix("10", models.Teacher, "class-1")
	if c := rows[0].Cells[0]; c.Status != models.ReviewSubmitted || c.SubmissionID == "" {
		t.Fatalf("handed-in cell = %+v", c)
	}
}
