package projects

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type UseCase interface {
	CreateProject(project *models.ProjectCore) (id string, err error)
	GetProjectById(projectId, userId string) (project *models.ProjectCore, err error)
	UpdateProject(project *models.ProjectCore, userId string) (err error)
}
