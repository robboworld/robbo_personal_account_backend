package usecase

import (
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

type stubCommentGateway struct {
	project  *models.ScratchProjectDB
	comment  *models.ScratchProjectCommentDB
	deleted  bool
	created  *models.ScratchProjectCommentDB
}

func (s *stubCommentGateway) GetScratchProjectById(string) (*models.ScratchProjectDB, error) {
	if s.project == nil {
		return nil, projectPage.ErrPageNotFound
	}
	return s.project, nil
}
func (s *stubCommentGateway) GetCommentByID(string) (*models.ScratchProjectCommentDB, error) {
	if s.comment == nil {
		return nil, projectPage.ErrCommentNotFound
	}
	return s.comment, nil
}
func (s *stubCommentGateway) SoftDeleteComment(string) error {
	s.deleted = true
	return nil
}
func (s *stubCommentGateway) CreateComment(c *models.ScratchProjectCommentDB) (*models.ScratchProjectCommentDB, error) {
	c.ID = "new-comment"
	c.CreatedAt = time.Now().UTC()
	s.created = c
	return c, nil
}
func (s *stubCommentGateway) ListProjectComments(string, int) ([]models.ScratchProjectCommentDB, error) {
	return nil, nil
}
func (s *stubCommentGateway) GetCommentReactionSummary(string, string) (*models.CommentReactionsHTTP, error) {
	return &models.CommentReactionsHTTP{}, nil
}
func (s *stubCommentGateway) GetCommentReactionSummaries([]string, string) (map[string]models.CommentReactionsHTTP, error) {
	return map[string]models.CommentReactionsHTTP{}, nil
}
func (s *stubCommentGateway) UpsertCommentReaction(string, string, string) error { return nil }
func (s *stubCommentGateway) DeleteCommentReaction(string, string) error         { return nil }

// Unused Gateway methods — embed nil panic stubs via panicking methods only if called.
func (s *stubCommentGateway) CreateProjectPage(*models.ProjectPageCore) (*models.ProjectPageCore, error) {
	panic("unused")
}
func (s *stubCommentGateway) UpdateProjectPage(*models.ProjectPageCore) (*models.ProjectPageCore, error) {
	panic("unused")
}
func (s *stubCommentGateway) DeleteProjectPage(string) error { panic("unused") }
func (s *stubCommentGateway) GetProjectPageById(string) (*models.ProjectPageCore, error) {
	panic("unused")
}
func (s *stubCommentGateway) GetProjectPageByProjectId(string) (*models.ProjectPageCore, error) {
	panic("unused")
}
func (s *stubCommentGateway) GetPublicProjectPages(int, int, projectPage.PublicListFilter) ([]*models.ProjectPageCore, int64, error) {
	panic("unused")
}
func (s *stubCommentGateway) GetPreviewImage(string) ([]byte, string, error) { panic("unused") }
func (s *stubCommentGateway) SavePreviewImage(string, []byte, string) error  { panic("unused") }
func (s *stubCommentGateway) GetLatestSb3Archive(string) ([]byte, error)     { panic("unused") }
func (s *stubCommentGateway) SaveSb3Archive(string, string, []byte, string) error {
	panic("unused")
}
func (s *stubCommentGateway) GetTotalStorageBytesForOwner(string) (int64, error) { panic("unused") }
func (s *stubCommentGateway) GetCurrentVersionSizeBytes(string) (int64, error)   { panic("unused") }
func (s *stubCommentGateway) ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error) {
	panic("unused")
}
func (s *stubCommentGateway) GetProjectReactionSummary(string, string) (*models.ProjectReactionsHTTP, error) {
	panic("unused")
}
func (s *stubCommentGateway) UpsertProjectReaction(string, string, string) error { panic("unused") }
func (s *stubCommentGateway) DeleteProjectReaction(string, string) error         { panic("unused") }
func (s *stubCommentGateway) SetLandingFeatured(string, bool, int) (*models.ProjectPageCore, error) {
	panic("unused")
}
func (s *stubCommentGateway) ReorderLandingFeatured([]projectPage.LandingFeaturedOrderItem) error {
	panic("unused")
}

func (s *stubCommentGateway) CountProjectsByOwner(string) (int64, error) {
	panic("unused")
}

func TestDeleteProjectComment_Author(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: true},
		comment: &models.ScratchProjectCommentDB{ID: "c1", ProjectID: "p1", UserID: "author"},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	if err := uc.DeleteProjectComment("p1", "c1", "author", models.Student); err != nil {
		t.Fatal(err)
	}
	if !gw.deleted {
		t.Fatal("expected soft delete")
	}
}

func TestDeleteProjectComment_Owner(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: true},
		comment: &models.ScratchProjectCommentDB{ID: "c1", ProjectID: "p1", UserID: "author"},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	if err := uc.DeleteProjectComment("p1", "c1", "owner", models.Student); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteProjectComment_SuperAdmin(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: true},
		comment: &models.ScratchProjectCommentDB{ID: "c1", ProjectID: "p1", UserID: "author"},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	if err := uc.DeleteProjectComment("p1", "c1", "admin", models.SuperAdmin); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteProjectComment_Forbidden(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: true},
		comment: &models.ScratchProjectCommentDB{ID: "c1", ProjectID: "p1", UserID: "author"},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	err := uc.DeleteProjectComment("p1", "c1", "stranger", models.Student)
	if err != auth.ErrNotAccess {
		t.Fatalf("expected ErrNotAccess, got %v", err)
	}
}

func TestCreateProjectComment_PrivateRejected(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: false},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	_, err := uc.CreateProjectComment("p1", "user", "hello", nil)
	if err != auth.ErrNotAccess {
		t.Fatalf("expected ErrNotAccess, got %v", err)
	}
}

func TestCreateProjectComment_Profanity(t *testing.T) {
	gw := &stubCommentGateway{
		project: &models.ScratchProjectDB{ID: "p1", OwnerUserID: "owner", IsPublic: true},
	}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}
	_, err := uc.CreateProjectComment("p1", "user", "это хуй", nil)
	if err != projectPage.ErrProfanityDetected {
		t.Fatalf("expected ErrProfanityDetected, got %v", err)
	}
}
