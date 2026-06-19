package projectPage

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type UseCase interface {
	CreateProjectPage(authorId string) (newProjectPage *models.ProjectPageCore, err error)
	UpdateProjectPage(projectPage *models.ProjectPageCore, authorId string) (projectPageUpdated *models.ProjectPageCore, err error)
	DeleteProjectPage(projectId string, authorId string) (err error)
	GetAllProjectPageByUserId(authorId string, page int, pageSize int) (projectPages []*models.ProjectPageCore, countRows int64, err error)
	GetPublicProjectPages(page int, pageSize int) (projectPages []*models.ProjectPageCore, countRows int64, err error)
	GetProjectPageById(projectPageId string, viewerId string) (projectPage *models.ProjectPageCore, err error)
	DownloadProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error)
	IssuePlayToken(projectPageId string, viewerId string, backendBaseURL string) (*PlayTokenHTTP, error)
	PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error)
	GetProjectJSONByToken(projectId string, token string) (json string, err error)
	UploadProjectSb3(projectPageId string, ownerId string, data []byte) error
}
