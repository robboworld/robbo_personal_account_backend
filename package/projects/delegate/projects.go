package delegate

import (
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

func (p *ProjectDelegateImpl) CreateProject(project *models.ProjectHTTP) (newProject *models.ProjectCore, err error) {
	projectCore := project.ToCore()
	return p.UseCase.CreateProject(projectCore)
}

func (p *ProjectDelegateImpl) DeleteProject() {

}

func (p *ProjectDelegateImpl) UpdateProject(project *models.ProjectHTTP) (err error) {
	projectCore := project.ToCore()
	return p.UseCase.UpdateProject(projectCore)
}

func (p *ProjectDelegateImpl) GetProjectById(projectId, userId string, role models.Role) (project models.ProjectHTTP, err error) {
	projectCore, err := p.UseCase.GetProjectById(projectId, userId, role)
	if err != nil {
		return
	}
	project.FromCore(projectCore)
	return
}
