package delegate

import (
	"log"
	"strconv"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"go.uber.org/fx"
)

type ProjectPageDelegateImpl struct {
	projectPage.UseCase
}

type ProjectPageDelegateModule struct {
	fx.Out
	projectPage.Delegate
}

func SetupProjectPageDelegate(usecase projectPage.UseCase) ProjectPageDelegateModule {
	return ProjectPageDelegateModule{
		Delegate: &ProjectPageDelegateImpl{
			UseCase: usecase,
		},
	}
}

func (p *ProjectPageDelegateImpl) CreateProjectPage(authorId string, locale string) (newProjectPage models.ProjectPageHTTP, err error) {
	newProjectPageCore, err := p.UseCase.CreateProjectPage(authorId, locale)
	if err != nil {
		log.Println(err)
		return
	}
	newProjectPage.FromCore(newProjectPageCore)
	return
}

func (p *ProjectPageDelegateImpl) DeleteProjectPage(projectId string, authorId string) (err error) {
	return p.UseCase.DeleteProjectPage(projectId, authorId)
}

func (p *ProjectPageDelegateImpl) UpdateProjectPage(projectPage *models.ProjectPageHTTP, authorId string) (projectPageUpdated models.ProjectPageHTTP, err error) {
	projectPageCore := projectPage.ToCore()
	projectPageUpdatedCore, err := p.UseCase.UpdateProjectPage(projectPageCore, authorId)
	if err != nil {
		log.Println(err)
		return
	}
	projectPageUpdated.FromCore(projectPageUpdatedCore)
	return
}

func (p *ProjectPageDelegateImpl) GetProjectPageById(projectPageId string, viewerId string) (projectPage models.ProjectPageHTTP, err error) {
	projectPageCore, err := p.UseCase.GetProjectPageById(projectPageId, viewerId)
	if err != nil {
		return
	}
	projectPage.FromCore(projectPageCore)
	return
}

func (p *ProjectPageDelegateImpl) DownloadProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error) {
	return p.UseCase.DownloadProjectSb3(projectPageId, viewerId)
}

func (p *ProjectPageDelegateImpl) PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error) {
	return p.UseCase.PlayProjectSb3(projectPageId, viewerId)
}

func (p *ProjectPageDelegateImpl) IssuePlayToken(projectPageId string, viewerId string, backendBaseURL string) (*projectPage.PlayTokenHTTP, error) {
	return p.UseCase.IssuePlayToken(projectPageId, viewerId, backendBaseURL)
}

func (p *ProjectPageDelegateImpl) PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error) {
	return p.UseCase.PlayProjectSb3ByToken(projectPageId, token)
}

func (p *ProjectPageDelegateImpl) GetProjectJSONByToken(projectId string, token string) (string, error) {
	return p.UseCase.GetProjectJSONByToken(projectId, token)
}

func (p *ProjectPageDelegateImpl) UploadProjectSb3(projectPageId string, ownerId string, data []byte) error {
	return p.UseCase.UploadProjectSb3(projectPageId, ownerId, data)
}

func (p *ProjectPageDelegateImpl) ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error) {
	return p.UseCase.ListEnabledReactionTypes()
}

func (p *ProjectPageDelegateImpl) GetProjectReactions(
	projectPageId string,
	viewerId string,
) (*models.ProjectReactionsHTTP, error) {
	return p.UseCase.GetProjectReactions(projectPageId, viewerId)
}

func (p *ProjectPageDelegateImpl) PutProjectReaction(
	projectPageId string,
	userId string,
	reactionCode string,
) (*models.ProjectReactionsHTTP, error) {
	return p.UseCase.PutProjectReaction(projectPageId, userId, reactionCode)
}

func (p *ProjectPageDelegateImpl) DeleteProjectReaction(
	projectPageId string,
	userId string,
) (*models.ProjectReactionsHTTP, error) {
	return p.UseCase.DeleteProjectReaction(projectPageId, userId)
}

func (p *ProjectPageDelegateImpl) GetAllProjectPagesByUserId(authorId, page, pageSize string) (
	projectPages []*models.ProjectPageHTTP,
	countRows int,
	err error,
) {
	pageInt32, parsePageErr := strconv.ParseInt(page, 10, 32)
	pageSizeInt32, parseSzErr := strconv.ParseInt(pageSize, 10, 32)
	if parsePageErr != nil || pageInt32 < 1 {
		pageInt32 = 1
	}
	if parseSzErr != nil || pageSizeInt32 < 1 {
		pageSizeInt32 = 10
	}
	projectPagesCore, countRowsInt64, err := p.UseCase.GetAllProjectPageByUserId(
		authorId,
		int(pageInt32),
		int(pageSizeInt32),
	)
	if err != nil {
		return
	}
	countRows = int(countRowsInt64)
	for _, projectPageCore := range projectPagesCore {
		var projectPageHttp models.ProjectPageHTTP
		projectPageHttp.FromCore(projectPageCore)
		projectPages = append(projectPages, &projectPageHttp)
	}
	return
}

func (p *ProjectPageDelegateImpl) GetPublicProjectPages(page, pageSize string, landingFeaturedOnly bool) (
	projectPages []*models.ProjectPageHTTP,
	countRows int,
	err error,
) {
	pageInt32, parsePageErr := strconv.ParseInt(page, 10, 32)
	pageSizeInt32, parseSzErr := strconv.ParseInt(pageSize, 10, 32)
	if parsePageErr != nil || pageInt32 < 1 {
		pageInt32 = 1
	}
	if parseSzErr != nil || pageSizeInt32 < 1 {
		pageSizeInt32 = 10
	}
	projectPagesCore, countRowsInt64, err := p.UseCase.GetPublicProjectPages(int(pageInt32), int(pageSizeInt32), landingFeaturedOnly)
	if err != nil {
		return
	}
	countRows = int(countRowsInt64)
	for _, projectPageCore := range projectPagesCore {
		var projectPageHttp models.ProjectPageHTTP
		projectPageHttp.FromCore(projectPageCore)
		projectPages = append(projectPages, &projectPageHttp)
	}
	return
}

func (p *ProjectPageDelegateImpl) GetPreviewImage(projectPageId string, viewerId string) (data []byte, mime string, err error) {
	return p.UseCase.GetPreviewImage(projectPageId, viewerId)
}

func (p *ProjectPageDelegateImpl) SavePreviewImage(projectPageId string, ownerId string, data []byte, mime string) error {
	return p.UseCase.SavePreviewImage(projectPageId, ownerId, data, mime)
}

func (p *ProjectPageDelegateImpl) ModerateDeleteProjectPage(projectPageId, moderatorId, reason string) error {
	return p.UseCase.ModerateDeleteProjectPage(projectPageId, moderatorId, reason)
}

func (p *ProjectPageDelegateImpl) SetLandingFeatured(
	projectPageId string,
	featured bool,
	sortOrder int,
) (models.ProjectPageHTTP, error) {
	var out models.ProjectPageHTTP
	core, err := p.UseCase.SetLandingFeatured(projectPageId, featured, sortOrder)
	if err != nil {
		return out, err
	}
	out.FromCore(core)
	return out, nil
}

func (p *ProjectPageDelegateImpl) ReorderLandingFeatured(items []projectPage.LandingFeaturedOrderItem) error {
	return p.UseCase.ReorderLandingFeatured(items)
}
