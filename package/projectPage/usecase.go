package projectPage

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// TeacherEditRecorder tells the class that a teacher changed handed-in work (restored: put the
// student's version back). A hook, like access.CohortTeacherReader: this package knows nothing
// about classes.
var TeacherEditRecorder func(teacherID string, work *models.ScratchProjectDB, restored bool)

// EditingState answers the editor's signal. Tracked is false once the viewer no longer counts as
// editing; Others are the rest of the people editing the same work.
type EditingState struct {
	Tracked bool                   `json:"tracked"`
	Others  []models.ProjectEditor `json:"others,omitempty"`
}

type UseCase interface {
	CreateProjectPage(authorId string, locale string) (newProjectPage *models.ProjectPageCore, err error)
	UpdateProjectPage(projectPage *models.ProjectPageCore, authorId string, role models.Role) (projectPageUpdated *models.ProjectPageCore, err error)
	DeleteProjectPage(projectId string, authorId string) (err error)
	GetAllProjectPageByUserId(authorId string, viewerId string, page int, pageSize int) (projectPages []*models.ProjectPageCore, countRows int64, err error)
	GetPublicProjectPages(page int, pageSize int, filter PublicListFilter) (projectPages []*models.ProjectPageCore, countRows int64, err error)
	GetProjectPageById(projectPageId string, viewerId string, role models.Role) (projectPage *models.ProjectPageCore, err error)
	GetPreviewImage(projectPageId string, viewerId string, role models.Role) (data []byte, mime string, err error)
	SavePreviewImage(projectPageId string, ownerId string, data []byte, mime string) error
	DownloadProjectSb3(projectPageId string, viewerId string, role models.Role) (data []byte, filename string, err error)
	PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	IssuePlayToken(projectPageId string, viewerId string, role models.Role, backendBaseURL string) (*PlayTokenHTTP, error)
	PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error)
	GetProjectJSONByToken(projectId string, token string) (json string, err error)
	UploadProjectSb3(projectPageId string, uploaderId string, data []byte) error
	RestoreStudentVersion(projectPageId string, teacherId string) error
	TouchEditing(projectPageId string, userId string) (*EditingState, error)
	MaxSb3UploadBytes(ownerId string) (int64, error)
	PruneSb3Versions(policy Sb3RetentionPolicy) (deleted int64, err error)
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
	SetLandingFeatured(projectPageId string, featured bool, sortOrder int) (*models.ProjectPageCore, error)
	ReorderLandingFeatured(items []LandingFeaturedOrderItem) error
}
