package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// Only what the student flows touch; anything else panics on the nil interface.
type studentGateway struct {
	teacherclass.Gateway
	projects map[string]*models.ScratchProjectDB
	reviewed string
}

func (g *studentGateway) GetInviteByID(string) (*models.ClassInviteDB, error) {
	return &models.ClassInviteDB{ID: "class-1", EdxCourseID: "course-v1:R+S+1", EdxCohortID: "7", OwnerTeacherID: "10"}, nil
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

func (g *studentGateway) UpdateReview(projectID, _ string, _ string, _ *time.Time) error {
	g.reviewed = projectID
	return nil
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
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1")},
		"x/797":  {ID: "own", OwnerUserID: "797"},
	}}
	uc := &UseCaseImpl{gw: gw}

	if err := uc.SubmitAssignment("797", "own"); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("own project err = %v, want ErrBadRequest", err)
	}
	if err := uc.SubmitAssignment("798", "p1"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("someone else's copy err = %v, want ErrForbidden", err)
	}
	if err := uc.SubmitAssignment("797", "p1"); err != nil || gw.reviewed != "p1" {
		t.Fatalf("submit err=%v reviewed=%q", err, gw.reviewed)
	}
}
