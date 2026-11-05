package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// editingGateway serves the live mark over the drafts fake: draft of 797, handed-in work of 798.
type editingGateway struct {
	*draftsGateway
	rows []models.ScratchProjectEditorDB
	err  error
}

func (g *editingGateway) ListActiveEditors(ids []string) ([]models.ScratchProjectEditorDB, error) {
	var out []models.ScratchProjectEditorDB
	for _, r := range g.rows {
		for _, id := range ids {
			if r.ProjectID == id {
				out = append(out, r)
			}
		}
	}
	return out, g.err
}

func editingClass(t *testing.T) (*UseCaseImpl, *editingGateway) {
	t.Helper()
	withoutLMSNames(t)
	gw := &editingGateway{draftsGateway: newDraftsGateway()}
	return &UseCaseImpl{gw: gw, notify: &workNotifier{}}, gw
}

func TestBoardShowsWhoIsEditing(t *testing.T) {
	uc, gw := editingClass(t)
	gw.rows = []models.ScratchProjectEditorDB{
		{ProjectID: "sent", UserID: "10", Role: models.EditorTeacher},
		// Never written for a draft, and a teacher must not get it even so.
		{ProjectID: "draft", UserID: "797", Role: models.EditorStudent},
	}
	subs, err := uc.ListSubmissions("10", models.Teacher, "class-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got := submissionOf(subs, "798").Editors; len(got) != 1 || got[0].Role != models.EditorTeacher {
		t.Fatalf("handed-in work: %+v", got)
	}
	if got := submissionOf(subs, "797").Editors; len(got) != 0 {
		t.Fatalf("draft: %+v", got)
	}
}

func TestBoardSurvivesFailedEditorsLookup(t *testing.T) {
	uc, gw := editingClass(t)
	gw.err = errors.New("db down")
	subs, err := uc.ListSubmissions("10", models.Teacher, "class-1", "a1")
	if err != nil || len(subs) != 2 {
		t.Fatalf("subs=%+v err=%v", subs, err)
	}
}

func TestStudentSeesOnlyTeachersEditing(t *testing.T) {
	uc, gw := editingClass(t)
	withCohortMembers(t, "798")
	gw.rows = []models.ScratchProjectEditorDB{
		{ProjectID: "sent", UserID: "10", Role: models.EditorTeacher},
		{ProjectID: "sent", UserID: "798", Role: models.EditorStudent},
	}
	list, err := uc.ListAssignments("798", models.Student, "class-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == "a1" && (len(a.MyEditors) != 1 || a.MyEditors[0].Role != models.EditorTeacher) {
			t.Fatalf("a1: %+v", a.MyEditors)
		}
		if a.ID != "a1" && len(a.MyEditors) != 0 {
			t.Fatalf("%s: %+v", a.ID, a.MyEditors)
		}
	}
}
