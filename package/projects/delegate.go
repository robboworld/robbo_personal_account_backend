package projects

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Delegate interface {
	GetProjectById(projectId, userId string) (project models.ProjectHTTP, err error)
}
