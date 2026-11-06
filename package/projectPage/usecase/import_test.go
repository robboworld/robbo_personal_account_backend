package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
)

type importPageGateway struct {
	projectPage.Gateway
	used        int64
	saveErr     error
	saved       []byte
	savedSource string
}

func (g *importPageGateway) GetTotalStorageBytesForOwner(string) (int64, error) { return g.used, nil }

func (g *importPageGateway) CreateProjectPage(page *models.ProjectPageCore) (*models.ProjectPageCore, error) {
	return &models.ProjectPageCore{ProjectPageId: page.ProjectId, ProjectId: page.ProjectId, Title: page.Title}, nil
}

func (g *importPageGateway) SaveSb3Archive(_ string, _ string, data []byte, source string) error {
	if g.saveErr != nil {
		return g.saveErr
	}
	g.saved, g.savedSource = data, source
	return nil
}

type importProjectsGateway struct {
	projects.Gateway
	created []*models.ProjectCore
	deleted []string
}

func (g *importProjectsGateway) CreateProject(project *models.ProjectCore) (string, error) {
	g.created = append(g.created, project)
	return "p1", nil
}

func (g *importProjectsGateway) DeleteProject(id string) error {
	g.deleted = append(g.deleted, id)
	return nil
}

func importUseCase(used int64) (*ProjectPageUseCaseImpl, *importPageGateway, *importProjectsGateway) {
	pages := &importPageGateway{used: used}
	projs := &importProjectsGateway{}
	lic := &quotaLicensingGateway{}
	return &ProjectPageUseCaseImpl{
		projectPageGateway: pages,
		projectGateway:     projs,
		licensingGateway:   lic,
		quota:              storage.NewQuota(lic, pages, nil),
	}, pages, projs
}

func testSb3(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("project.json")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte(emptyProjectJson2))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImportProjectSb3CreatesNamedProject(t *testing.T) {
	uc, pages, projs := importUseCase(0)
	data := testSb3(t)
	created, err := uc.ImportProjectSb3("1", "  Моя игра  ", "ru", data)
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != "Моя игра" || !created.IsOwner {
		t.Fatalf("created=%+v, want title %q owned by the author", created, "Моя игра")
	}
	if len(projs.created) != 1 || projs.created[0].Json != emptyProjectJson2 || projs.created[0].AuthorId != "1" {
		t.Fatalf("project not created from the archive's project.json: %+v", projs.created)
	}
	if !bytes.Equal(pages.saved, data) || pages.savedSource != "lk.import" {
		t.Fatalf("archive not saved: %d bytes, source %q", len(pages.saved), pages.savedSource)
	}
}

func TestImportProjectSb3DefaultsAndCapsTitle(t *testing.T) {
	uc, _, _ := importUseCase(0)
	created, err := uc.ImportProjectSb3("1", " ", "en", testSb3(t))
	if err != nil || created.Title != "Untitled" {
		t.Fatalf("blank title: created=%+v err=%v, want Untitled", created, err)
	}
	created, err = uc.ImportProjectSb3("1", strings.Repeat("я", 300), "ru", testSb3(t))
	if err != nil || len([]rune(created.Title)) != maxImportTitleRunes {
		t.Fatalf("long title: %d runes err=%v, want %d", len([]rune(created.Title)), err, maxImportTitleRunes)
	}
}

func TestImportProjectSb3RejectsBeforeCreating(t *testing.T) {
	uc, _, projs := importUseCase(0)
	if _, err := uc.ImportProjectSb3("1", "x", "ru", []byte("not a zip")); !errors.Is(err, projectPage.ErrInvalidProjectFile) {
		t.Fatalf("invalid file: err=%v", err)
	}
	// Free: 100 MB in total, the upload would go past it.
	full, _, fullProjs := importUseCase(100 * mb)
	if _, err := full.ImportProjectSb3("1", "x", "ru", testSb3(t)); !errors.Is(err, projectPage.ErrStorageQuotaExceeded) {
		t.Fatalf("quota used up: err=%v", err)
	}
	if len(projs.created)+len(fullProjs.created) != 0 {
		t.Fatal("a rejected file must not leave a project behind")
	}
}

func TestImportProjectSb3RollsBackWhenArchiveNotSaved(t *testing.T) {
	uc, pages, projs := importUseCase(0)
	pages.saveErr = errors.New("db down")
	if _, err := uc.ImportProjectSb3("1", "x", "ru", testSb3(t)); err == nil {
		t.Fatal("expected the save error")
	}
	if len(projs.deleted) != 1 || projs.deleted[0] != "p1" {
		t.Fatalf("created project not rolled back: deleted=%v", projs.deleted)
	}
}
