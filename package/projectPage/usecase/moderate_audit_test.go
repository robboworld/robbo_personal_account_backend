package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
)

type moderatePages struct{ projectPage.Gateway }

func (moderatePages) GetScratchProjectById(id string) (*models.ScratchProjectDB, error) {
	return &models.ScratchProjectDB{ID: id, OwnerUserID: "7", Title: "Кот"}, nil
}
func (moderatePages) GetCurrentVersionSizeBytes(string) (int64, error) { return 42, nil }

type moderateProjects struct {
	projects.Gateway
	deleted []string
}

func (p *moderateProjects) DeleteProject(id string) error {
	p.deleted = append(p.deleted, id)
	return nil
}

type moderateAudit struct {
	rows []models.AdminContentAuditDB
	fail bool
}

func (a *moderateAudit) Write(e *models.AdminContentAuditDB) error {
	if a.fail {
		return errors.New("db down")
	}
	a.rows = append(a.rows, *e)
	return nil
}
func (a *moderateAudit) ListForTarget(string, int) ([]models.AdminContentAuditDB, error) {
	return nil, nil
}

func TestModerateDeleteWritesAuditFirst(t *testing.T) {
	projs := &moderateProjects{}
	au := &moderateAudit{}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: moderatePages{}, projectGateway: projs, audit: au}
	if err := uc.ModerateDeleteProjectPage("p1", "1", " спам "); err != nil {
		t.Fatal(err)
	}
	if len(projs.deleted) != 1 || len(au.rows) != 1 {
		t.Fatalf("deleted=%v audit=%v", projs.deleted, au.rows)
	}
	r := au.rows[0]
	if r.ActorUserID != "1" || r.TargetUserID != "7" || r.Action != models.AuditProjectDelete || r.SizeBytes != 42 || r.Reason != "спам" {
		t.Fatalf("audit row=%+v", r)
	}

	failing := &moderateProjects{}
	uc = &ProjectPageUseCaseImpl{projectPageGateway: moderatePages{}, projectGateway: failing, audit: &moderateAudit{fail: true}}
	if err := uc.ModerateDeleteProjectPage("p1", "1", "спам"); err == nil || len(failing.deleted) != 0 {
		t.Fatalf("deleted without an audit record: err=%v", err)
	}
	if err := uc.ModerateDeleteProjectPage("p1", "1", "a"); !errors.Is(err, projectPage.ErrBadRequest) {
		t.Fatalf("short reason: %v", err)
	}
}
