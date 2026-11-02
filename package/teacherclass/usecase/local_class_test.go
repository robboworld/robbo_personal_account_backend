package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// localGateway keeps classes and the ЛК roster in memory. The use case has no edx client,
// so any Open edX call from a ЛК class panics the test.
type localGateway struct {
	teacherclass.Gateway
	invites map[string]*models.ClassInviteDB
	members map[string]map[string]bool
}

func newLocalGateway() *localGateway {
	return &localGateway{invites: map[string]*models.ClassInviteDB{}, members: map[string]map[string]bool{}}
}

func (g *localGateway) CreateInvite(inv *models.ClassInviteDB) error {
	g.invites[inv.ID] = inv
	return nil
}

func (g *localGateway) GetInviteByID(id string) (*models.ClassInviteDB, error) {
	if inv, ok := g.invites[id]; ok {
		return inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *localGateway) GetInviteBySlug(slug string) (*models.ClassInviteDB, error) {
	for _, inv := range g.invites {
		if inv.InviteSlug == slug {
			return inv, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *localGateway) ListActiveInvites() ([]models.ClassInviteDB, error) {
	out := []models.ClassInviteDB{}
	for _, inv := range g.invites {
		out = append(out, *inv)
	}
	return out, nil
}

func (g *localGateway) AddMember(classID, userID string) error {
	if g.members[classID] == nil {
		g.members[classID] = map[string]bool{}
	}
	g.members[classID][userID] = true
	return nil
}

func (g *localGateway) RemoveMember(classID, userID string) error {
	delete(g.members[classID], userID)
	return nil
}

func (g *localGateway) ListMembers(classID string) ([]models.ClassMemberDB, error) {
	out := []models.ClassMemberDB{}
	for id := range g.members[classID] {
		out = append(out, models.ClassMemberDB{ClassID: classID, UserID: id})
	}
	return out, nil
}

func (g *localGateway) ListMemberClassIDs(userID string) ([]string, error) {
	var out []string
	for classID, m := range g.members {
		if m[userID] {
			out = append(out, classID)
		}
	}
	return out, nil
}

func (g *localGateway) IsMemberOfAny(userID string, classIDs []string) (bool, error) {
	for _, id := range classIDs {
		if g.members[id][userID] {
			return true, nil
		}
	}
	return false, nil
}

func (g *localGateway) ListAssignments(string, string) ([]models.ClassAssignmentDB, error) {
	return nil, nil
}

func withoutLMSNames(t *testing.T) {
	t.Helper()
	prev := ownerDisplayName
	ownerDisplayName = func(string) string { return "" }
	t.Cleanup(func() { ownerDisplayName = prev })
}

func TestLocalClassNeedsNoCourse(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	uc := &UseCaseImpl{gw: gw}

	dto, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок", LinkLMS: false})
	if err != nil {
		t.Fatal(err)
	}
	inv := gw.invites[dto.ID]
	if inv == nil || inv.LmsLinked || inv.EdxCourseID != "" || inv.EdxCohortID != "lk:"+dto.ID {
		t.Fatalf("invite = %+v", inv)
	}
	if dto.LmsLinked {
		t.Fatal("dto says linked")
	}
}

func TestLinkedClassStillNeedsCourse(t *testing.T) {
	uc := &UseCaseImpl{gw: newLocalGateway()}
	if _, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "5A", LinkLMS: true}); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

func TestLocalClassRosterStaysInLK(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	uc := &UseCaseImpl{gw: gw}
	dto, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uc.JoinClass("797", dto.InviteSlug); err != nil {
		t.Fatal(err)
	}
	if !gw.members[dto.ID]["797"] {
		t.Fatal("student not on the ЛК roster")
	}
	if _, err := uc.ListAssignments("797", models.Student, dto.ID); err != nil {
		t.Fatalf("member err = %v", err)
	}
	if _, err := uc.ListAssignments("798", models.Student, dto.ID); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("non-member err = %v, want ErrForbidden", err)
	}
	if err := uc.RemoveMember("10", models.Teacher, dto.ID, "797"); err != nil {
		t.Fatal(err)
	}
	if gw.members[dto.ID]["797"] {
		t.Fatal("student still on the roster")
	}
}

func TestStudentSeesLocalClassesWithoutLMS(t *testing.T) {
	withoutLMSNames(t)
	prev := studentCohortIDs
	studentCohortIDs = func(string) (map[string]bool, error) { return nil, errors.New("lms down") }
	t.Cleanup(func() { studentCohortIDs = prev })

	gw := newLocalGateway()
	gw.invites["cohort"] = &models.ClassInviteDB{ID: "cohort", EdxCohortID: "7", LmsLinked: true}
	gw.invites["local"] = &models.ClassInviteDB{ID: "local", EdxCohortID: "lk:local"}
	gw.members["local"] = map[string]bool{"797": true}
	uc := &UseCaseImpl{gw: gw}

	list, err := uc.ListStudentClasses("797")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "local" {
		t.Fatalf("classes = %+v, want only local", list)
	}
}

// assignGateway: a ЛК class whose student "bad" cannot get a copy.
type assignGateway struct {
	*localGateway
	forked    []string
	templates []*models.ScratchProjectDB
}

func (g *assignGateway) GetProject(id string) (*models.ScratchProjectDB, error) {
	return &models.ScratchProjectDB{ID: id, OwnerUserID: "10"}, nil
}

func (g *assignGateway) CreateAssignment(a *models.ClassAssignmentDB) error {
	a.ID = "a1"
	return nil
}

func (g *assignGateway) ForkProjectForStudent(tpl *models.ScratchProjectDB, studentUserID, _ string, _ *models.ClassAssignmentDB) (*models.ScratchProjectDB, error) {
	if studentUserID == "bad" {
		return nil, errors.New("insert failed")
	}
	g.forked = append(g.forked, studentUserID)
	g.templates = append(g.templates, tpl)
	return &models.ScratchProjectDB{}, nil
}

func TestCreateAssignmentReportsCopies(t *testing.T) {
	withoutLMSNames(t)
	gw := &assignGateway{localGateway: newLocalGateway()}
	uc := &UseCaseImpl{gw: gw}
	class, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"797", "798", "bad"} {
		if err := gw.AddMember(class.ID, id); err != nil {
			t.Fatal(err)
		}
	}

	dto, err := uc.CreateAssignment("10", models.Teacher, class.ID, teacherclass.CreateAssignmentInput{
		Title: "Maze", AssignmentKey: "maze", TemplateProjectID: "tpl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dto.CopiesIssued != 2 || dto.CopiesFailed != 1 || dto.CopiesSkipped != 0 {
		t.Fatalf("copies issued=%d failed=%d skipped=%d, want 2/1/0", dto.CopiesIssued, dto.CopiesFailed, dto.CopiesSkipped)
	}
}

func TestCreateAssignmentWithoutTemplateIssuesEmptyProjects(t *testing.T) {
	withoutLMSNames(t)
	gw := &assignGateway{localGateway: newLocalGateway()}
	uc := &UseCaseImpl{gw: gw}
	class, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gw.AddMember(class.ID, "797"); err != nil {
		t.Fatal(err)
	}

	dto, err := uc.CreateAssignment("10", models.Teacher, class.ID, teacherclass.CreateAssignmentInput{
		Title: "Free", AssignmentKey: "free",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dto.TemplateProjectID != nil {
		t.Fatalf("template = %q, want none", *dto.TemplateProjectID)
	}
	if dto.CopiesIssued != 1 || len(gw.templates) != 1 || gw.templates[0] != nil {
		t.Fatalf("issued=%d templates=%v, want one empty project", dto.CopiesIssued, gw.templates)
	}
}
