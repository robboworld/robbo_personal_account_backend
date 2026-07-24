package projectPage

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Gateway interface {
	CreateProjectPage(projectPageCore *models.ProjectPageCore) (newProjectPage *models.ProjectPageCore, err error)
	UpdateProjectPage(projectPageCore *models.ProjectPageCore) (projectPageUpdated *models.ProjectPageCore, err error)
	DeleteProjectPage(projectId string) (err error)
	GetProjectPageById(projectPageId string) (projectPageCore *models.ProjectPageCore, err error)
	GetProjectPageByProjectId(projectId string) (projectPageCore *models.ProjectPageCore, err error)
	GetScratchProjectById(projectPageId string) (projectDB *models.ScratchProjectDB, err error)
	GetPublicProjectPages(page, pageSize int, landingFeaturedOnly bool) (projectPages []*models.ProjectPageCore, countRows int64, err error)
	GetPreviewImage(projectPageId string) (data []byte, mime string, err error)
	SavePreviewImage(projectPageId string, data []byte, mime string) error
	GetLatestSb3Archive(projectPageId string) (archive []byte, err error)
	SaveSb3Archive(projectPageId, userID string, archive []byte, saveSource string) error
	ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error)
	GetProjectReactionSummary(projectId, viewerUserId string) (*models.ProjectReactionsHTTP, error)
	UpsertProjectReaction(projectId, userId, reactionCode string) error
	DeleteProjectReaction(projectId, userId string) error
	SetLandingFeatured(projectPageId string, featured bool, sortOrder int) (*models.ProjectPageCore, error)
	ReorderLandingFeatured(items []LandingFeaturedOrderItem) error
}

// LandingFeaturedOrderItem is one row for batch showcase reorder.
type LandingFeaturedOrderItem struct {
	ProjectPageID string
	SortOrder     int
}
