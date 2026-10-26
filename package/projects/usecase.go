package projects

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type UseCase interface {
	GetProjectById(projectId, userId string) (project *models.ProjectCore, err error)
}
