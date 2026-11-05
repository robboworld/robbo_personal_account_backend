package usecase

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// Only what the student flows touch; anything else panics on the nil interface.
type studentGateway struct {
	teacherclass.Gateway
	projects    map[string]*models.ScratchProjectDB
	assignments map[string]*models.ClassAssignmentDB
	// unsaved lists copies the student never saved.
	unsaved   map[string]bool
	deleted   []string
	deleteErr error
	reviewed  string
	reviewer  string
	newStatus string
	handIn    *teacherclass.HandIn
}

// GetAssignment: an entry of assignments, else an open assignment of cohort 7.
func (g *studentGateway) GetAssignment(id string) (*models.ClassAssignmentDB, error) {
	if a, ok := g.assignments[id]; ok {
		return a, nil
	}
	return &models.ClassAssignmentDB{ID: id, EdxCohortID: "7"}, nil
}

func (g *studentGateway) SetAssignmentClosed(id string, closed bool) error {
	a, _ := g.GetAssignment(id)
	a.ClosedAt = nil
	if closed {
		now := time.Now()
		a.ClosedAt = &now
	}
	if g.assignments == nil {
		g.assignments = map[string]*models.ClassAssignmentDB{}
	}
	g.assignments[id] = a
	return nil
}

func (g *studentGateway) DeleteAssignment(id string) error {
	if g.deleteErr != nil {
		return g.deleteErr
	}
	g.deleted = append(g.deleted, id)
	return nil
}

func (g *studentGateway) GetInviteByID(string) (*models.ClassInviteDB, error) {
	return &models.ClassInviteDB{ID: "class-1", EdxCourseID: "course-v1:R+S+1", EdxCohortID: "7", OwnerTeacherID: "10", LmsLinked: true}, nil
}

func (g *studentGateway) GetInviteByCourseCohort(string, string) (*models.ClassInviteDB, error) {
	return g.GetInviteByID("class-1")
}

func (g *studentGateway) ListAssignments(string, string) ([]models.ClassAssignmentDB, error) {
	return []models.ClassAssignmentDB{{ID: "a1", Title: "Maze"}, {ID: "a2", Title: "Cat"}}, nil
}

func (g *studentGateway) FindStudentAssignmentProject(assignmentID, studentID string) (*models.ScratchProjectDB, error) {
	if p, ok := g.projects[assignmentID+"/"+studentID]; ok {
		return p, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *studentGateway) GetProject(id string) (*models.ScratchProjectDB, error) {
	for _, p := range g.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *studentGateway) HasOwnSave(id string) (bool, error) {
	return !g.unsaved[id], nil
}

func (g *studentGateway) UpdateReview(projectID, status, _, reviewerID string, handIn *teacherclass.HandIn) error {
	g.reviewed = projectID
	g.reviewer = reviewerID
	g.newStatus = status
	g.handIn = handIn
	return nil
}

func (g *studentGateway) IsCoTeacher(string, string) (bool, error)  { return false, nil }
func (g *studentGateway) WasCoTeacher(string, string) (bool, error) { return false, nil }
func (g *studentGateway) ListCoTeachers(string) ([]models.ClassCoTeacherDB, error) {
	return nil, nil
}
func (g *studentGateway) ListCoTeachings(string) ([]models.ClassCoTeacherDB, error) {
	return nil, nil
}

func strPtr(s string) *string { return &s }

func withCohortMembers(t *testing.T, members ...string) {
	t.Helper()
	prev := studentInCohort
	studentInCohort = func(userID, cohortID string) (bool, error) {
		for _, m := range members {
			if m == userID && cohortID == "7" {
				return true, nil
			}
		}
		return false, nil
	}
	t.Cleanup(func() { studentInCohort = prev })
}

func TestStudentAssignmentsRequireMembership(t *testing.T) {
	withCohortMembers(t, "797")
	uc := &UseCaseImpl{gw: &studentGateway{}}

	if _, err := uc.ListAssignments("798", models.Student, "class-1"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("non-member err = %v, want ErrForbidden", err)
	}
}

func TestStudentAssignmentsCarryOwnProject(t *testing.T) {
	withCohortMembers(t, "797")
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ReviewStatus: strPtr("returned"), ReviewComment: strPtr("Add a score")},
	}}
	uc := &UseCaseImpl{gw: gw}

	list, err := uc.ListAssignments("797", models.Student, "class-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d assignments", len(list))
	}
	if list[0].MyProjectID != "p1" || list[0].MyReviewStatus != "returned" || list[0].MyReviewComment != "Add a score" {
		t.Fatalf("assignment a1 = %+v", list[0])
	}
	if list[1].MyProjectID != "" {
		t.Fatalf("assignment a2 has no copy yet, got %q", list[1].MyProjectID)
	}
}

func TestSubmitOnlyAssignmentCopies(t *testing.T) {
	withCohortMembers(t, "797")
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1")},
		"x/797":  {ID: "own", OwnerUserID: "797"},
	}}
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}

	if err := uc.SubmitAssignment("797", "own", ""); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("own project err = %v, want ErrBadRequest", err)
	}
	if err := uc.SubmitAssignment("798", "p1", ""); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("someone else's copy err = %v, want ErrForbidden", err)
	}
	if err := uc.SubmitAssignment("797", "p1", ""); err != nil || gw.reviewed != "p1" {
		t.Fatalf("submit err=%v reviewed=%q", err, gw.reviewed)
	}
}

func TestSubmitCarriesTheStudentNote(t *testing.T) {
	withCohortMembers(t, "797")
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1")},
	}}
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}

	// The limit is in characters: 2000 Cyrillic letters are 4000 bytes.
	tooLong := strings.Repeat("я", maxSubmissionNoteChars+1)
	for _, bad := range []string{tooLong, "a\x00b"} {
		if err := uc.SubmitAssignment("797", "p1", bad); !errors.Is(err, teacherclass.ErrBadRequest) {
			t.Fatalf("note of %d bytes: err = %v, want ErrBadRequest", len(bad), err)
		}
		if gw.reviewed != "" {
			t.Fatal("a rejected note must not hand the work in")
		}
	}
	longest := strings.Repeat("я", maxSubmissionNoteChars)
	for in, want := range map[string]string{"  кот ходит по сцене\n": "кот ходит по сцене", longest: longest, "": ""} {
		if err := uc.SubmitAssignment("797", "p1", in); err != nil {
			t.Fatal(err)
		}
		if gw.handIn == nil || gw.handIn.Note != want || gw.handIn.At.IsZero() {
			t.Fatalf("hand-in = %+v, want the note %q", gw.handIn, want)
		}
	}
}

func (g *studentGateway) SetProjectNote(projectID, note string) error {
	for _, p := range g.projects {
		if p.ID == projectID && deref(p.ReviewStatus) == models.ReviewSubmitted {
			p.SubmissionNote = note
			return nil
		}
	}
	return teacherclass.ErrNotOnReview
}

func TestUpdateProjectNote(t *testing.T) {
	withCohortMembers(t, "797")
	own := &models.ScratchProjectDB{ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"),
		ReviewStatus: strPtr(models.ReviewSubmitted), SubmissionNote: "было"}
	gw := &studentGateway{
		projects: map[string]*models.ScratchProjectDB{"a1/797": own, "x/797": {ID: "own", OwnerUserID: "797"}},
		assignments: map[string]*models.ClassAssignmentDB{
			"a1": {ID: "a1", EdxCourseID: "c1", EdxCohortID: "1"},
		},
	}
	uc := &UseCaseImpl{gw: gw}

	if err := uc.UpdateProjectNote("798", "p1", "чужое"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("someone else's work: %v", err)
	}
	if err := uc.UpdateProjectNote("797", "own", "x"); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("not an assignment copy: %v", err)
	}
	if err := uc.UpdateProjectNote("797", "p1", strings.Repeat("я", maxSubmissionNoteChars+1)); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("over the limit: %v", err)
	}
	if err := uc.UpdateProjectNote("797", "p1", "  стало\n"); err != nil || own.SubmissionNote != "стало" {
		t.Fatalf("edit: err=%v note=%q", err, own.SubmissionNote)
	}
	if err := uc.UpdateProjectNote("797", "p1", ""); err != nil || own.SubmissionNote != "" {
		t.Fatalf("removing the text: err=%v note=%q", err, own.SubmissionNote)
	}
	if gw.reviewed != "" {
		t.Fatal("a note edit must not touch the review")
	}

	closed := time.Now()
	gw.assignments["a1"].ClosedAt = &closed
	if err := uc.UpdateProjectNote("797", "p1", "x"); !errors.Is(err, teacherclass.ErrAssignmentClosed) {
		t.Fatalf("closed assignment: %v", err)
	}
	gw.assignments["a1"].ClosedAt = nil
	own.ReviewStatus = strPtr(models.ReviewAccepted)
	if err := uc.UpdateProjectNote("797", "p1", "x"); !errors.Is(err, teacherclass.ErrNotOnReview) {
		t.Fatalf("accepted work: %v", err)
	}
}

func TestSubmitOnlyDraftOrReturned(t *testing.T) {
	withCohortMembers(t, "797")
	for status, want := range map[string]error{
		models.ReviewDraft:        nil,
		models.ReviewReturned:     nil,
		models.ReviewSubmitted:    teacherclass.ErrAlreadyHandedIn,
		models.ReviewAccepted:     teacherclass.ErrAlreadyHandedIn,
		models.ReviewShownInClass: teacherclass.ErrAlreadyHandedIn,
	} {
		gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
			"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ReviewStatus: strPtr(status)},
		}}
		uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}
		if err := uc.SubmitAssignment("797", "p1", ""); !errors.Is(err, want) {
			t.Fatalf("submit from %s: err = %v, want %v", status, err, want)
		}
		if want != nil && gw.reviewed != "" {
			t.Fatalf("submit from %s changed the review", status)
		}
	}
}

func TestSubmitClosedAssignment(t *testing.T) {
	withCohortMembers(t, "797")
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ReviewStatus: strPtr(models.ReviewDraft)},
	}}
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}

	if _, err := uc.SetAssignmentClosed("10", models.Teacher, "class-1", "a1", true); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitAssignment("797", "p1", ""); !errors.Is(err, teacherclass.ErrAssignmentClosed) {
		t.Fatalf("closed: err = %v, want ErrAssignmentClosed", err)
	}
	if _, err := uc.SetAssignmentClosed("10", models.Teacher, "class-1", "a1", false); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitAssignment("797", "p1", ""); err != nil {
		t.Fatalf("reopened: err = %v", err)
	}
}

func TestSubmitAfterDueIsAllowed(t *testing.T) {
	withCohortMembers(t, "797")
	past := time.Now().Add(-time.Hour)
	gw := &studentGateway{
		projects: map[string]*models.ScratchProjectDB{
			"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ReviewStatus: strPtr(models.ReviewDraft)},
		},
		assignments: map[string]*models.ClassAssignmentDB{"a1": {ID: "a1", EdxCohortID: "7", DueAt: &past}},
	}
	if err := (&UseCaseImpl{gw: gw, notify: &workNotifier{}}).SubmitAssignment("797", "p1", ""); err != nil {
		t.Fatalf("late submit err = %v", err)
	}
}

func TestSubmitNeedsMembership(t *testing.T) {
	withCohortMembers(t)
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ReviewStatus: strPtr(models.ReviewDraft)},
	}}
	if err := (&UseCaseImpl{gw: gw}).SubmitAssignment("797", "p1", ""); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("left the class: err = %v, want ErrForbidden", err)
	}
	if gw.reviewed != "" {
		t.Fatal("work of a student who left went to review")
	}
}

func TestCloseOtherTeachersAssignment(t *testing.T) {
	uc := &UseCaseImpl{gw: &studentGateway{}}
	if _, err := uc.SetAssignmentClosed("11", models.Teacher, "class-1", "a1", true); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

type classListGateway struct {
	studentGateway
}

func (g *classListGateway) ListActiveInvites() ([]models.ClassInviteDB, error) {
	return []models.ClassInviteDB{
		{ID: "c7", EdxCohortID: "7", DisplayName: "5A", LmsLinked: true},
		{ID: "c9", EdxCohortID: "9", DisplayName: "6B", LmsLinked: true},
	}, nil
}

func (g *classListGateway) ListMemberClassIDs(string) ([]string, error) { return nil, nil }

func TestStudentClassesComeFromCohortMembership(t *testing.T) {
	prev := studentCohortIDs
	studentCohortIDs = func(string) (map[string]bool, error) { return map[string]bool{"9": true}, nil }
	t.Cleanup(func() { studentCohortIDs = prev })
	uc := &UseCaseImpl{gw: &classListGateway{}}

	list, err := uc.ListStudentClasses("797")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "c9" {
		t.Fatalf("classes = %+v, want only c9", list)
	}
}
