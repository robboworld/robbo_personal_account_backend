package projects

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Delegate interface {
	CreateProject(project *models.ProjectHTTP) (id string, err error)
	GetProjectById(projectId, userId string) (project models.ProjectHTTP, err error)
	UpdateProject(project *models.ProjectHTTP, userId string) (err error)
}
