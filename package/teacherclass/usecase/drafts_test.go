package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// draftsGateway: class-1 (teacher 10, cohort 7) where student 797 has a draft of a1
// and student 798 submitted a1.
type draftsGateway struct {
	studentGateway
}

func newDraftsGateway() *draftsGateway {
	cohort := "7"
	return &draftsGateway{studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "draft", OwnerUserID: "797", EdxCohortID: &cohort, AssignmentID: strPtr("a1"), ReviewStatus: strPtr(models.ReviewDraft)},
		"a1/798": {ID: "sent", OwnerUserID: "798", EdxCohortID: &cohort, AssignmentID: strPtr("a1"), ReviewStatus: strPtr(models.ReviewSubmitted)},
	}}}
}

func (g *draftsGateway) ListSubmissions(assignmentID string) ([]models.ScratchProjectDB, error) {
	var out []models.ScratchProjectDB
	for _, p := range g.projects {
		if *p.AssignmentID == assignmentID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (g *draftsGateway) ListSubmissionsForAssignments(ids []string) ([]models.ScratchProjectDB, error) {
	var out []models.ScratchProjectDB
	for _, id := range ids {
		list, _ := g.ListSubmissions(id)
		out = append(out, list...)
	}
	return out, nil
}

func (g *draftsGateway) GetInviteByCourseCohort(string, string) (*models.ClassInviteDB, error) {
	return g.GetInviteByID("class-1")
}

func submissionIDs(list []teacherclass.SubmissionDTO) map[string]bool {
	ids := map[string]bool{}
	for _, s := range list {
		ids[s.ProjectID] = true
	}
	return ids
}

func submissionOf(list []teacherclass.SubmissionDTO, owner string) *teacherclass.SubmissionDTO {
	for i := range list {
		if list[i].OwnerUserID == owner {
			return &list[i]
		}
	}
	return nil
}

func TestTeacherSeesDraftsWithoutContent(t *testing.T) {
	uc := &UseCaseImpl{gw: newDraftsGateway()}

	subs, err := uc.ListSubmissions("10", models.Teacher, "class-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if ids := submissionIDs(subs); ids["draft"] || !ids["sent"] {
		t.Fatalf("teacher submission ids = %v, want only the submitted one", ids)
	}
	draft := submissionOf(subs, "797")
	if draft == nil || draft.ReviewStatus != models.ReviewDraft || draft.PreviewURL != "" {
		t.Fatalf("teacher draft = %+v, want status without project", draft)
	}

	_, rows, err := uc.ProgressMatrix("10", models.Teacher, "class-1")
	if err != nil {
		t.Fatal(err)
	}
	var draftCell bool
	for _, r := range rows {
		for _, c := range r.Cells {
			if c.ProjectID == "draft" {
				t.Fatalf("teacher progress links the draft: %+v", r)
			}
			if r.UserID == "797" && c.Status == models.ReviewDraft {
				draftCell = true
			}
		}
	}
	if !draftCell {
		t.Fatalf("teacher progress = %+v, want a draft cell for 797", rows)
	}

	if err := uc.ReviewSubmission("10", models.Teacher, "draft", models.ReviewAccepted, ""); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("review draft err = %v, want ErrNotFound", err)
	}
	if err := uc.ReviewSubmission("10", models.Teacher, "sent", models.ReviewAccepted, ""); err != nil {
		t.Fatalf("review submitted err = %v", err)
	}
}

func TestTeacherCannotSetDraft(t *testing.T) {
	uc := &UseCaseImpl{gw: newDraftsGateway()}
	if err := uc.ReviewSubmission("10", models.Teacher, "sent", models.ReviewDraft, ""); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("review to draft err = %v, want ErrBadRequest", err)
	}
	if err := uc.ReviewSubmission("10", models.Teacher, "sent", models.ReviewSubmitted, ""); err != nil {
		t.Fatalf("back to review queue err = %v", err)
	}
}

func TestLateSubmission(t *testing.T) {
	gw := newDraftsGateway()
	due := time.Now().Add(-time.Hour)
	submitted := time.Now()
	gw.projects["a1/798"].SubmittedAt = &submitted
	gw.assignments = map[string]*models.ClassAssignmentDB{"a1": {ID: "a1", EdxCohortID: "7", DueAt: &due}}
	uc := &UseCaseImpl{gw: gw}

	subs, err := uc.ListSubmissions("10", models.Teacher, "class-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if s := submissionOf(subs, "798"); s == nil || !s.Late {
		t.Fatalf("submission = %+v, want late", s)
	}
}

func TestSuperAdminSeesDrafts(t *testing.T) {
	uc := &UseCaseImpl{gw: newDraftsGateway()}

	subs, err := uc.ListSubmissions("1", models.SuperAdmin, "class-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if ids := submissionIDs(subs); !ids["draft"] || !ids["sent"] {
		t.Fatalf("super admin submissions = %v, want both", ids)
	}
	if err := uc.ReviewSubmission("1", models.SuperAdmin, "draft", models.ReviewReturned, ""); err != nil {
		t.Fatalf("super admin review draft err = %v", err)
	}
}
