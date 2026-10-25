package usecase

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
)

// Only the methods the author list touches; any other call panics on the nil interface.
type authorListPageGateway struct {
	projectPage.Gateway
	publicFilter   *projectPage.PublicListFilter
	publicPageSize int
}

func (g *authorListPageGateway) GetPublicProjectPages(page, pageSize int, filter projectPage.PublicListFilter) ([]*models.ProjectPageCore, int64, error) {
	g.publicFilter = &filter
	g.publicPageSize = pageSize
	return []*models.ProjectPageCore{{ProjectPageId: "shared-1", IsShared: true}}, 1, nil
}

func (g *authorListPageGateway) GetScratchProjectById(id string) (*models.ScratchProjectDB, error) {
	return &models.ScratchProjectDB{ID: id, OwnerUserID: "798", IsPublic: true}, nil
}

func (g *authorListPageGateway) GetProjectPageByProjectId(id string) (*models.ProjectPageCore, error) {
	return &models.ProjectPageCore{ProjectPageId: id}, nil
}

type authorListProjectsGateway struct {
	projects.Gateway
	called bool
}

func (g *authorListProjectsGateway) GetProjectsByAuthorId(string, int, int) ([]*models.ProjectCore, int64, error) {
	g.called = true
	return []*models.ProjectCore{{ID: "private-1"}}, 1, nil
}

func newAuthorListUseCase() (*ProjectPageUseCaseImpl, *authorListPageGateway, *authorListProjectsGateway) {
	pages := &authorListPageGateway{}
	projs := &authorListProjectsGateway{}
	return &ProjectPageUseCaseImpl{projectPageGateway: pages, projectGateway: projs}, pages, projs
}

func TestAuthorListForOtherViewerShowsOnlySharedProjects(t *testing.T) {
	uc, pages, projs := newAuthorListUseCase()

	list, _, err := uc.GetAllProjectPageByUserId("798", "797", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if projs.called {
		t.Fatal("another viewer must not reach the author's full (private) project list")
	}
	if pages.publicFilter == nil || pages.publicFilter.OwnerUserID != "798" {
		t.Fatalf("public list filter = %+v, want OwnerUserID=798", pages.publicFilter)
	}
	for _, p := range list {
		if p.IsOwner {
			t.Fatalf("project %s marked IsOwner for a non-owner", p.ProjectPageId)
		}
	}
}

func TestAuthorListForGuestShowsOnlySharedProjects(t *testing.T) {
	uc, _, projs := newAuthorListUseCase()

	if _, _, err := uc.GetAllProjectPageByUserId("798", "", 1, 10); err != nil {
		t.Fatal(err)
	}
	if projs.called {
		t.Fatal("a guest must not reach the author's full project list")
	}
}

func TestAuthorListForOwnerShowsAllProjects(t *testing.T) {
	uc, pages, projs := newAuthorListUseCase()

	list, _, err := uc.GetAllProjectPageByUserId("798", "798", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !projs.called || pages.publicFilter != nil {
		t.Fatal("the owner must get the full list, not the public one")
	}
	if len(list) != 1 || !list[0].IsOwner {
		t.Fatalf("owner list = %+v, want one project with IsOwner", list)
	}
}

func TestAuthorListCapsPageSize(t *testing.T) {
	uc, pages, _ := newAuthorListUseCase()

	if _, _, err := uc.GetAllProjectPageByUserId("798", "797", 1, 100000); err != nil {
		t.Fatal(err)
	}
	if pages.publicPageSize != maxAuthorListPageSize {
		t.Fatalf("pageSize = %d, want %d", pages.publicPageSize, maxAuthorListPageSize)
	}
}
