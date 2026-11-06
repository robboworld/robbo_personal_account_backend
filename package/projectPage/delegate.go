package projectPage

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Delegate interface {
	CreateProjectPage(authorId string, locale string) (newProjectPage models.ProjectPageHTTP, err error)
	ImportProjectSb3(authorId string, title string, locale string, data []byte) (newProjectPage models.ProjectPageHTTP, err error)
	UpdateProjectPage(projectPage *models.ProjectPageHTTP, authorId string, role models.Role) (projectPageUpdated models.ProjectPageHTTP, err error)
	DeleteProjectPage(projectId string, authorId string) (err error)
	GetProjectPageById(projectPageId string, viewerId string, role models.Role) (projectPage models.ProjectPageHTTP, err error)
	GetAllProjectPagesByUserId(authorId string, viewerId string, page string, pageSize string) (projectPages []*models.ProjectPageHTTP, countRows int, err error)
	GetPublicProjectPages(page string, pageSize string, filter PublicListFilter) (projectPages []*models.ProjectPageHTTP, countRows int, err error)
	DownloadProjectSb3(projectPageId string, viewerId string, role models.Role) (data []byte, filename string, err error)
	PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	IssuePlayToken(projectPageId string, viewerId string, role models.Role, backendBaseURL string) (*PlayTokenHTTP, error)
	PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error)
	GetProjectJSONByToken(projectId string, token string) (json string, err error)
	UploadProjectSb3(projectPageId string, uploaderId string, data []byte) error
	RestoreStudentVersion(projectPageId string, teacherId string) error
	TouchEditing(projectPageId string, userId string) (*EditingState, error)
	MaxSb3UploadBytes(ownerId string) (int64, error)
	GetPreviewImage(projectPageId string, viewerId string, role models.Role) (data []byte, mime string, err error)
	SavePreviewImage(projectPageId string, ownerId string, data []byte, mime string) error
	ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error)
	GetProjectReactions(projectPageId, viewerId string, role models.Role) (*models.ProjectReactionsHTTP, error)
	PutProjectReaction(projectPageId, userId, reactionCode string) (*models.ProjectReactionsHTTP, error)
	DeleteProjectReaction(projectPageId, userId string) (*models.ProjectReactionsHTTP, error)
	GetProjectComments(projectPageId, viewerId string) (*models.ProjectCommentsHTTP, error)
	CreateProjectComment(projectPageId, userId, body string, parentID *string) (*models.ProjectCommentHTTP, error)
	DeleteProjectComment(projectPageId, commentId, userId string, role models.Role) error
	GetCommentReactions(projectPageId, commentId, viewerId string) (*models.CommentReactionsHTTP, error)
	PutCommentReaction(projectPageId, commentId, userId, reactionCode string) (*models.CommentReactionsHTTP, error)
	DeleteCommentReaction(projectPageId, commentId, userId string) (*models.CommentReactionsHTTP, error)
	ModerateDeleteProjectPage(projectPageId, moderatorId, reason string) error
	SetLandingFeatured(projectPageId string, featured bool, sortOrder int) (models.ProjectPageHTTP, error)
	ReorderLandingFeatured(items []LandingFeaturedOrderItem) error
}
