package projects

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Gateway interface {
	CreateProject(project *models.ProjectCore) (newProject *models.ProjectCore, err error)
	DeleteProject(projectId string) (err error)
	GetProjectById(projectId string) (project *models.ProjectCore, err error)
	GetProjectsByAuthorId(authorId string, page, pageSize int) (
		projects []*models.ProjectCore,
		countRows int64,
		err error,
	)
	UpdateProject(project *models.ProjectCore) (err error)
}
