package delegate

import (
	"fmt"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	"go.uber.org/fx"
)

type ProjectDelegateImpl struct {
	projects.UseCase
}

type ProjectDelegateModule struct {
	fx.Out
	projects.Delegate
}

func SetupProjectDelegate(usecase projects.UseCase) ProjectDelegateModule {
	return ProjectDelegateModule{
		Delegate: &ProjectDelegateImpl{usecase},
	}
}

func (p *ProjectDelegateImpl) GetProjectById(projectId, userId string) (project models.ProjectHTTP, err error) {
	projectCore, err := p.UseCase.GetProjectById(projectId, userId)
	fmt.Print(projectCore)
	project.FromCore(projectCore)
	return
}
