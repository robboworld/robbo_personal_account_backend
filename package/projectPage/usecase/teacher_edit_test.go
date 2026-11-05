package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/access"
)

const editTeacher = "teacher-1"

// editPageGateway adds a teacher's writes to the fake of the homework lock tests.
type editPageGateway struct {
	*lockPageGateway
	saveErr error
	kept    []byte
	touched []string
	editors []models.ScratchProjectEditorDB
}

func (g *editPageGateway) SaveTeacherEdit(string, string, []byte) error {
	if g.saveErr != nil {
		return g.saveErr
	}
	g.writes = append(g.writes, "teacher sb3")
	return nil
}

func (g *editPageGateway) RestoreStudentVersion(string) ([]byte, error) {
	if g.saveErr != nil {
		return nil, g.saveErr
	}
	g.writes = append(g.writes, "restore")
	return g.kept, nil
}

func (g *editPageGateway) TouchProjectEditor(projectID, userID, role string) error {
	g.touched = append(g.touched, userID+":"+role)
	return nil
}

func (g *editPageGateway) ListProjectEditors(string) ([]models.ScratchProjectEditorDB, error) {
	return g.editors, nil
}

type recordedEdit struct {
	teacher  string
	restored bool
}

// editUseCase serves an assignment copy of a class that editTeacher runs.
func editUseCase(t *testing.T, status string) (*ProjectPageUseCaseImpl, *editPageGateway, *[]recordedEdit) {
	t.Helper()
	assignment, cohort := "a-1", "lk:class-1"
	uc, lock := lockUseCase(&assignment, status)
	lock.row.EdxCohortID = &cohort
	lock.row.ScratchVMJSON = emptyProjectJson2
	pages := &editPageGateway{lockPageGateway: lock}
	uc.projectPageGateway = pages

	var edits []recordedEdit
	savedReader, savedRecorder := access.CohortTeacherReader, projectPage.TeacherEditRecorder
	t.Cleanup(func() {
		access.CohortTeacherReader, projectPage.TeacherEditRecorder = savedReader, savedRecorder
	})
	access.CohortTeacherReader = func(id, _, _ string) bool { return id == editTeacher }
	projectPage.TeacherEditRecorder = func(teacher string, _ *models.ScratchProjectDB, restored bool) {
		edits = append(edits, recordedEdit{teacher, restored})
	}
	return uc, pages, &edits
}

// editedSb3 is a project with a costume name changed: another project.json than emptyProjectJson2.
func editedSb3(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("project.json")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte(strings.Replace(emptyProjectJson2, "costume1", "учитель", 1)))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTeacherSavesWorkOnReview(t *testing.T) {
	for _, status := range []string{models.ReviewSubmitted, models.ReviewAccepted, models.ReviewShownInClass} {
		uc, pages, edits := editUseCase(t, status)
		if err := uc.UploadProjectSb3("p1", editTeacher, editedSb3(t)); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		if strings.Join(pages.writes, ",") != "teacher sb3,json" {
			t.Fatalf("%s: writes = %v", status, pages.writes)
		}
		if len(*edits) != 1 || (*edits)[0] != (recordedEdit{teacher: editTeacher}) {
			t.Fatalf("%s: edits = %+v", status, *edits)
		}
	}
}

// The editor writes a new .sb3 on every save: pressing Save must not tell the student anything.
func TestTeacherSaveWithoutChangeWritesNothing(t *testing.T) {
	uc, pages, edits := editUseCase(t, models.ReviewSubmitted)
	if err := uc.UploadProjectSb3("p1", editTeacher, lockSb3(t)); err != nil {
		t.Fatal(err)
	}
	if len(pages.writes) != 0 || len(*edits) != 0 {
		t.Fatalf("writes = %v, edits = %+v", pages.writes, *edits)
	}
}

func TestTeacherDoesNotSaveReturnedWork(t *testing.T) {
	uc, pages, edits := editUseCase(t, models.ReviewReturned)
	if err := uc.UploadProjectSb3("p1", editTeacher, editedSb3(t)); !errors.Is(err, projectPage.ErrWorkNotWithTeacher) {
		t.Fatalf("err = %v, want ErrWorkNotWithTeacher", err)
	}
	if err := uc.RestoreStudentVersion("p1", editTeacher); !errors.Is(err, projectPage.ErrWorkNotWithTeacher) {
		t.Fatalf("restore err = %v, want ErrWorkNotWithTeacher", err)
	}
	if len(pages.writes) != 0 || len(*edits) != 0 {
		t.Fatalf("writes = %v, edits = %+v", pages.writes, *edits)
	}
	// Someone else stays 403: the code must not reveal the review state of a stranger's work.
	if err := uc.UploadProjectSb3("p1", "teacher-2", editedSb3(t)); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("stranger err = %v, want ErrNotAccess", err)
	}
}

func TestDraftStaysClosedToTheTeacher(t *testing.T) {
	uc, _, _ := editUseCase(t, models.ReviewDraft)
	if err := uc.UploadProjectSb3("p1", editTeacher, editedSb3(t)); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("err = %v, want ErrNotAccess", err)
	}
}

// The work went back to the student between the check and the write.
func TestTeacherSaveRacingReturnTellsNobody(t *testing.T) {
	uc, pages, edits := editUseCase(t, models.ReviewSubmitted)
	pages.saveErr = projectPage.ErrWorkNotWithTeacher
	if err := uc.UploadProjectSb3("p1", editTeacher, editedSb3(t)); !errors.Is(err, projectPage.ErrWorkNotWithTeacher) {
		t.Fatalf("err = %v", err)
	}
	if len(*edits) != 0 {
		t.Fatalf("edits = %+v", *edits)
	}
}

func TestTeacherEditKeepsLockAndFields(t *testing.T) {
	uc, pages, _ := editUseCase(t, models.ReviewSubmitted)
	// The owner stays locked out while a teacher can write.
	if err := uc.UploadProjectSb3("p1", lockStudent, editedSb3(t)); !errors.Is(err, projectPage.ErrHomeworkLocked) {
		t.Fatalf("owner err = %v", err)
	}
	// Blocks only: the title and the description stay the student's, and so does the project.
	if _, err := uc.UpdateProjectPage(&models.ProjectPageCore{ProjectPageId: "p1", Title: "Другое"}, editTeacher, models.Teacher); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("title err = %v", err)
	}
	if err := uc.DeleteProjectPage("p1", editTeacher); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("delete err = %v", err)
	}
	if len(pages.writes) != 0 {
		t.Fatalf("writes = %v", pages.writes)
	}
}

func TestTeacherSavesPreviewOfWorkOnReview(t *testing.T) {
	uc, pages, _ := editUseCase(t, models.ReviewSubmitted)
	if err := uc.SavePreviewImage("p1", editTeacher, []byte{1}, "image/png"); err != nil {
		t.Fatal(err)
	}
	if err := uc.SavePreviewImage("p1", "teacher-2", []byte{1}, "image/png"); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("stranger err = %v", err)
	}
	if strings.Join(pages.writes, ",") != "preview" {
		t.Fatalf("writes = %v", pages.writes)
	}
}

func TestRestoreStudentVersion(t *testing.T) {
	uc, pages, edits := editUseCase(t, models.ReviewAccepted)
	pages.kept = lockSb3(t)
	if err := uc.RestoreStudentVersion("p1", "teacher-2"); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("stranger err = %v", err)
	}
	if err := uc.RestoreStudentVersion("p1", lockStudent); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("owner err = %v", err)
	}
	if err := uc.RestoreStudentVersion("p1", editTeacher); err != nil {
		t.Fatal(err)
	}
	if strings.Join(pages.writes, ",") != "restore,json" {
		t.Fatalf("writes = %v", pages.writes)
	}
	if len(*edits) != 1 || (*edits)[0] != (recordedEdit{editTeacher, true}) {
		t.Fatalf("edits = %+v", *edits)
	}
}

func TestProjectPageTellsAboutTeacherEdit(t *testing.T) {
	uc, pages, _ := editUseCase(t, models.ReviewSubmitted)
	at := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	pages.row.TeacherEditedAt = &at
	pages.row.IsPublic = true

	page, err := uc.GetProjectPageById("p1", editTeacher, models.Teacher)
	if err != nil || !page.TeacherCanEdit || page.TeacherEditedAt == "" {
		t.Fatalf("teacher: err=%v page=%+v", err, page)
	}
	page, err = uc.GetProjectPageById("p1", lockStudent, models.Student)
	if err != nil || page.TeacherCanEdit || page.TeacherEditedAt == "" {
		t.Fatalf("owner: err=%v page=%+v", err, page)
	}
	// A public copy: a stranger reads the project, not what happened in the class.
	page, err = uc.GetProjectPageById("p1", "someone", models.Student)
	if err != nil || page.TeacherCanEdit || page.TeacherEditedAt != "" {
		t.Fatalf("stranger: err=%v page=%+v", err, page)
	}
}

func TestTeacherEditingSignal(t *testing.T) {
	uc, pages, _ := editUseCase(t, models.ReviewSubmitted)
	pages.editors = []models.ScratchProjectEditorDB{
		{UserID: editTeacher, Role: models.EditorTeacher},
		{UserID: "teacher-2", Role: models.EditorTeacher},
	}
	state, err := uc.TouchEditing("p1", editTeacher)
	if err != nil || !state.Tracked {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if strings.Join(pages.touched, ",") != editTeacher+":teacher" {
		t.Fatalf("touched = %v", pages.touched)
	}
	// Oneself is not a reason for a warning.
	if len(state.Others) != 1 || state.Others[0].Role != models.EditorTeacher {
		t.Fatalf("others = %+v", state.Others)
	}
}

func TestEditingSignalCountsOnlyWhoMaySave(t *testing.T) {
	cases := []struct {
		name, status, viewer string
		tracked              bool
	}{
		{"student on returned work", models.ReviewReturned, lockStudent, true},
		{"student on locked work", models.ReviewSubmitted, lockStudent, false},
		{"student on a draft", models.ReviewDraft, lockStudent, false},
		{"teacher on returned work", models.ReviewReturned, editTeacher, false},
		{"teacher of another class", models.ReviewSubmitted, "teacher-2", false},
	}
	for _, tc := range cases {
		uc, pages, _ := editUseCase(t, tc.status)
		state, err := uc.TouchEditing("p1", tc.viewer)
		if err != nil || state.Tracked != tc.tracked {
			t.Errorf("%s: state=%+v err=%v", tc.name, state, err)
		}
		if (len(pages.touched) == 1) != tc.tracked {
			t.Errorf("%s: touched = %v", tc.name, pages.touched)
		}
	}
}

func TestProjectPageNamesEditorsToTheClassOnly(t *testing.T) {
	uc, pages, _ := editUseCase(t, models.ReviewReturned)
	pages.row.IsPublic = true
	pages.editors = []models.ScratchProjectEditorDB{
		{UserID: editTeacher, Role: models.EditorTeacher},
		{UserID: lockStudent, Role: models.EditorStudent},
	}
	page, err := uc.GetProjectPageById("p1", editTeacher, models.Teacher)
	if err != nil || len(page.Editors) != 2 {
		t.Fatalf("teacher: err=%v editors=%+v", err, page.Editors)
	}
	page, err = uc.GetProjectPageById("p1", lockStudent, models.Student)
	if err != nil || len(page.Editors) != 1 || page.Editors[0].Role != models.EditorTeacher || !page.EditingTracked {
		t.Fatalf("owner: err=%v page=%+v", err, page)
	}
	page, err = uc.GetProjectPageById("p1", "someone", models.Student)
	if err != nil || len(page.Editors) != 0 || page.EditingTracked {
		t.Fatalf("stranger: err=%v page=%+v", err, page)
	}
}
