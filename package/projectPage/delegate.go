package projectPage

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Delegate interface {
	CreateProjectPage(authorId string) (newProjectPage models.ProjectPageHTTP, err error)
	UpdateProjectPage(projectPage *models.ProjectPageHTTP, authorId string) (projectPageUpdated models.ProjectPageHTTP, err error)
	DeleteProjectPage(projectId string, authorId string) (err error)
	GetProjectPageById(projectPageId string, authorId string) (projectPage models.ProjectPageHTTP, err error)
	GetAllProjectPagesByUserId(authorId string, page string, pageSize string) (projectPages []*models.ProjectPageHTTP, countRows int, err error)
	DownloadProjectSb3(projectPageId string, authorId string) (data []byte, filename string, err error)
}
