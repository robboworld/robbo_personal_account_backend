package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
)

// lockPageGateway serves one assignment copy and records every write that reaches storage.
type lockPageGateway struct {
	projectPage.Gateway
	row     *models.ScratchProjectDB
	writes  []string
	updated *models.ProjectPageCore
}

func (g *lockPageGateway) GetScratchProjectById(string) (*models.ScratchProjectDB, error) {
	return g.row, nil
}
func (g *lockPageGateway) GetProjectPageById(id string) (*models.ProjectPageCore, error) {
	return &models.ProjectPageCore{ProjectPageId: id, ProjectId: id, Title: "Задание", Tags: []string{}}, nil
}
func (g *lockPageGateway) UpdateProjectPage(page *models.ProjectPageCore) (*models.ProjectPageCore, error) {
	g.writes = append(g.writes, "fields")
	g.updated = page
	return page, nil
}
func (g *lockPageGateway) SavePreviewImage(string, []byte, string) error {
	g.writes = append(g.writes, "preview")
	return nil
}
func (g *lockPageGateway) SaveSb3Archive(string, string, []byte, string) error {
	g.writes = append(g.writes, "sb3")
	return nil
}
func (g *lockPageGateway) GetCurrentVersionSizeBytes(string) (int64, error) { return 0, nil }
func (g *lockPageGateway) ListProjectEditors(string) ([]models.ScratchProjectEditorDB, error) {
	return nil, nil
}
func (g *lockPageGateway) PruneSb3Versions(string, projectPage.Sb3RetentionPolicy) (int64, error) {
	return 0, nil
}

type lockProjectsGateway struct {
	projects.Gateway
	pages *lockPageGateway
}

func (g *lockProjectsGateway) UpdateProject(*models.ProjectCore) error {
	g.pages.writes = append(g.pages.writes, "json")
	return nil
}
func (g *lockProjectsGateway) DeleteProject(string) error {
	g.pages.writes = append(g.pages.writes, "delete")
	return nil
}

const lockStudent = "student-1"

func lockUseCase(assignmentID *string, status string) (*ProjectPageUseCaseImpl, *lockPageGateway) {
	row := &models.ScratchProjectDB{ID: "p1", OwnerUserID: lockStudent, AssignmentID: assignmentID}
	if status != "" {
		row.ReviewStatus = &status
	}
	pages := &lockPageGateway{row: row}
	return &ProjectPageUseCaseImpl{
		projectPageGateway: pages,
		projectGateway:     &lockProjectsGateway{pages: pages},
	}, pages
}

func lockSb3(t *testing.T) []byte {
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

// ownerWrites runs every write the owner can make to a project page and returns their errors.
func ownerWrites(t *testing.T, uc *ProjectPageUseCaseImpl) map[string]error {
	t.Helper()
	_, updateErr := uc.UpdateProjectPage(&models.ProjectPageCore{ProjectPageId: "p1", Title: "Другое"}, lockStudent, models.Student)
	return map[string]error{
		"upload":  uc.UploadProjectSb3("p1", lockStudent, lockSb3(t)),
		"preview": uc.SavePreviewImage("p1", lockStudent, []byte{1}, "image/png"),
		"update":  updateErr,
		"delete":  uc.DeleteProjectPage("p1", lockStudent),
	}
}

func TestHandedInHomeworkIsReadOnlyForOwner(t *testing.T) {
	assignment := "a-1"
	for _, status := range []string{models.ReviewSubmitted, models.ReviewAccepted, models.ReviewShownInClass} {
		uc, pages := lockUseCase(&assignment, status)
		for op, err := range ownerWrites(t, uc) {
			if !errors.Is(err, projectPage.ErrHomeworkLocked) {
				t.Errorf("%s: %s err=%v, want ErrHomeworkLocked", status, op, err)
			}
		}
		if len(pages.writes) != 0 {
			t.Errorf("%s: writes reached storage: %v", status, pages.writes)
		}
	}
}

func TestOwnerEditsDraftReturnedAndOwnProjects(t *testing.T) {
	assignment := "a-1"
	cases := map[string]*ProjectPageUseCaseImpl{}
	cases["draft copy"], _ = lockUseCase(&assignment, models.ReviewDraft)
	cases["copy with no status"], _ = lockUseCase(&assignment, "")
	cases["returned copy"], _ = lockUseCase(&assignment, models.ReviewReturned)
	cases["own project"], _ = lockUseCase(nil, "")
	for name, uc := range cases {
		for op, err := range ownerWrites(t, uc) {
			if err != nil {
				t.Errorf("%s: %s err=%v", name, op, err)
			}
		}
	}
}

func TestHomeworkLockKeepsOtherRules(t *testing.T) {
	assignment := "a-1"
	uc, pages := lockUseCase(&assignment, models.ReviewSubmitted)

	// Someone else's project stays 403, not a lock message that reveals the review state.
	if err := uc.UploadProjectSb3("p1", "student-2", lockSb3(t)); !errors.Is(err, auth.ErrNotAccess) {
		t.Fatalf("stranger upload err=%v, want ErrNotAccess", err)
	}
	// A SuperAdmin still edits tags (only tags) on others' projects.
	_, err := uc.UpdateProjectPage(&models.ProjectPageCore{ProjectPageId: "p1", Title: "Другое", Tags: []string{"robots"}}, "admin-1", models.SuperAdmin)
	if err != nil || pages.updated == nil || pages.updated.Title != "Задание" {
		t.Fatalf("SuperAdmin tags edit: err=%v updated=%+v", err, pages.updated)
	}
}

func TestReviewLockedReachesProjectPage(t *testing.T) {
	assignment := "a-1"
	uc, _ := lockUseCase(&assignment, models.ReviewAccepted)
	page, err := uc.GetProjectPageById("p1", lockStudent, models.Student)
	if err != nil || !page.ReviewLocked {
		t.Fatalf("accepted copy: err=%v reviewLocked=%v", err, page != nil && page.ReviewLocked)
	}
	uc, _ = lockUseCase(&assignment, models.ReviewReturned)
	if page, _ = uc.GetProjectPageById("p1", lockStudent, models.Student); page.ReviewLocked {
		t.Fatal("returned copy reported as locked")
	}
}
