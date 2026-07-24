package projectPage

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Delegate interface {
	CreateProjectPage(authorId string, locale string) (newProjectPage models.ProjectPageHTTP, err error)
	UpdateProjectPage(projectPage *models.ProjectPageHTTP, authorId string) (projectPageUpdated models.ProjectPageHTTP, err error)
	DeleteProjectPage(projectId string, authorId string) (err error)
	GetProjectPageById(projectPageId string, viewerId string) (projectPage models.ProjectPageHTTP, err error)
	GetAllProjectPagesByUserId(authorId string, page string, pageSize string) (projectPages []*models.ProjectPageHTTP, countRows int, err error)
	GetPublicProjectPages(page string, pageSize string, landingFeaturedOnly bool) (projectPages []*models.ProjectPageHTTP, countRows int, err error)
	DownloadProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	IssuePlayToken(projectPageId string, viewerId string, backendBaseURL string) (*PlayTokenHTTP, error)
	PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error)
	GetProjectJSONByToken(projectId string, token string) (json string, err error)
	UploadProjectSb3(projectPageId string, ownerId string, data []byte) error
	GetPreviewImage(projectPageId string, viewerId string) (data []byte, mime string, err error)
	SavePreviewImage(projectPageId string, ownerId string, data []byte, mime string) error
	ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error)
	GetProjectReactions(projectPageId, viewerId string) (*models.ProjectReactionsHTTP, error)
	PutProjectReaction(projectPageId, userId, reactionCode string) (*models.ProjectReactionsHTTP, error)
	DeleteProjectReaction(projectPageId, userId string) (*models.ProjectReactionsHTTP, error)
	ModerateDeleteProjectPage(projectPageId, moderatorId, reason string) error
	SetLandingFeatured(projectPageId string, featured bool, sortOrder int) (models.ProjectPageHTTP, error)
	ReorderLandingFeatured(items []LandingFeaturedOrderItem) error
}
