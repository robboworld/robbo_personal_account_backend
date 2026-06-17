package usecase

import (
	"strings"
	"unicode"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type ProjectPageUseCaseImpl struct {
	projectPageGateway projectPage.Gateway
	projectGateway     projects.Gateway
}

type ProjectPageUseCaseModule struct {
	fx.Out
	projectPage.UseCase
}

func SetupProjectPageUseCase(projectPageGateway projectPage.Gateway, projectGateway projects.Gateway) ProjectPageUseCaseModule {
	return ProjectPageUseCaseModule{
		UseCase: &ProjectPageUseCaseImpl{
			projectPageGateway: projectPageGateway,
			projectGateway:     projectGateway,
		},
	}
}

const emptyProjectJson2 = "{\"targets\":[{\"isStage\":true,\"name\":\"Stage\",\"variables\":{\"`jEk@4|i[#Fk?(8x)AV." +
	"-my variable\":[\"my variable\",0]},\"lists\":{},\"broadcasts\":{},\"blocks\":{},\"currentCostume\":0,\"costum" +
	"es\":[{\"assetId\":\"cd21514d0531fdffb22204e0ec5ed84a\",\"name\":\"backdrop1\",\"md5ext\":\"cd21514d0531fdffb2" +
	"2204e0ec5ed84a.svg\",\"dataFormat\":\"svg\",\"rotationCenterX\":240,\"rotationCenterY\":180}],\"sounds\":[{\"a" +
	"ssetId\":\"83a9787d4cb6f3b7632b4ddfebf74367\",\"name\":\"pop\",\"dataFormat\":\"wav\",\"format\":\"\",\"rate\"" +
	":11025,\"sampleCount\":258,\"md5ext\":\"83a9787d4cb6f3b7632b4ddfebf74367.wav\"}],\"volume\":100},{\"isStage\":" +
	"false,\"name\":\"Sprite1\",\"variables\":{},\"lists\":{},\"broadcasts\":{},\"blocks\":{},\"currentCostume\":0," +
	"\"costumes\":[{\"assetId\":\"f8e72b8244738d0b448e46b38c5db6c2\",\"name\":\"costume1\",\"md5ext\":\"f8e72b82447" +
	"38d0b448e46b38c5db6c2.svg\",\"dataFormat\":\"svg\",\"bitmapResolution\":1,\"rotationCenterX\":67,\"rotationCen" +
	"terY\":95},{\"assetId\":\"bb3a866c4db08353f6faf43c54990f10\",\"name\":\"costume2\",\"md5ext\":\"bb3a866c4db083" +
	"53f6faf43c54990f10.svg\",\"dataFormat\":\"svg\",\"bitmapResolution\":1,\"rotationCenterX\":67,\"rotationCenter" +
	"Y\":95},{\"assetId\":\"bb6b82c9fa7c432c552ca2f251ae2078\",\"name\":\"costume3\",\"md5ext\":\"bb6b82c9fa7c432c5" +
	"52ca2f251ae2078.svg\",\"dataFormat\":\"svg\",\"bitmapResolution\":1,\"rotationCenterX\":67,\"rotationCenterY\"" +
	":95},{\"assetId\":\"be2345a4417ff516f9c1a5ece86a8c64\",\"name\":\"costume4\",\"md5ext\":\"be2345a4417ff516f9c1" +
	"a5ece86a8c64.svg\",\"dataFormat\":\"svg\",\"bitmapResolution\":1,\"rotationCenterX\":67,\"rotationCenterY\":95" +
	"}],\"sounds\":[{\"assetId\":\"28c76b6bebd04be1383fe9ba4933d263\",\"name\":\"Beep\",\"dataFormat\":\"wav\",\"fo" +
	"rmat\":\"\",\"rate\":11025,\"sampleCount\":9536,\"md5ext\":\"28c76b6bebd04be1383fe9ba4933d263.wav\"}],\"volume" +
	"\":100,\"visible\":true,\"x\":0,\"y\":0,\"size\":100,\"direction\":90,\"draggable\":false,\"rotationStyle\":\"" +
	"all around\"}],\"meta\":{\"semver\":\"3.0.0\",\"vm\":\"0.1.0\",\"agent\":\"Mozilla/5.0 (Macintosh; Intel Mac O" +
	"S X 10_13_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36\"}}"

const emptyProjectJson = "{\"targets\":[{\"isStage\":true,\"name\":\"Stage\",\"variables\":{\"`jEk@4|i[#Fk?(8x)AV." +
	"-my variable\":[\"my variable\",0]},\"lists\":{},\"broadcasts\":{},\"blocks\":{},\"comments\":{},\"currentCos" +
	"tume\":0,\"costumes\":[{\"assetId\":\"cd21514d0531fdffb22204e0ec5ed84a\",\"name\":\"backdrop1\",\"md5ext\":\"" +
	"cd21514d0531fdffb22204e0ec5ed84a.svg\",\"dataFormat\":\"svg\",\"rotationCenterX\":240,\"rotationCenterY\":180" +
	"}],\"sounds\":[{\"assetId\":\"83a9787d4cb6f3b7632b4ddfebf74367\",\"name\":\"pop\",\"dataFormat\":\"wav\",\"fo" +
	"rmat\":\"\",\"rate\":44100,\"sampleCount\":1032,\"md5ext\":\"83a9787d4cb6f3b7632b4ddfebf74367.wav\"}],\"volum" +
	"e\":100,\"layerOrder\":0,\"tempo\":60,\"videoTransparency\":50,\"videoState\":\"on\",\"textToSpeechLanguage\"" +
	":null},{\"isStage\":false,\"name\":\"Sprite1\",\"variables\":{},\"lists\":{},\"broadcasts\":{},\"blocks\":{}," +
	"\"comments\":{},\"currentCostume\":0,\"costumes\":[{\"assetId\":\"bcf454acf82e4504149f7ffe07081dbc\",\"name\"" +
	":\"costume1\",\"bitmapResolution\":1,\"md5ext\":\"bcf454acf82e4504149f7ffe07081dbc.svg\",\"dataFormat\":\"svg" +
	"\",\"rotationCenterX\":48,\"rotationCenterY\":50},{\"assetId\":\"0fb9be3e8397c983338cb71dc84d0b25\",\"name\":" +
	"\"costume2\",\"bitmapResolution\":1,\"md5ext\":\"0fb9be3e8397c983338cb71dc84d0b25.svg\",\"dataFormat\":\"svg\"" +
	",\"rotationCenterX\":46,\"rotationCenterY\":53}],\"sounds\":[{\"assetId\":\"83c36d806dc92327b9e7049a565c6bff\"" +
	",\"name\":\"Meow\",\"dataFormat\":\"wav\",\"format\":\"\",\"rate\":44100,\"sampleCount\":37376,\"md5ext\":\"8" +
	"3c36d806dc92327b9e7049a565c6bff.wav\"}],\"volume\":100,\"layerOrder\":1,\"visible\":true,\"x\":0,\"y\":0,\"si" +
	"ze\":100,\"direction\":90,\"draggable\":false,\"rotationStyle\":\"all around\"}],\"monitors\":[],\"extensions" +
	"\":[],\"meta\":{\"semver\":\"3.0.0\",\"vm\":\"0.2.0-prerelease.20220519142410\",\"agent\":\"Mozilla/5.0 (X11;" +
	" Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36\"}}"

func (p *ProjectPageUseCaseImpl) CreateProjectPage(authorId string) (newProjectPage *models.ProjectPageCore, err error) {
	project := models.ProjectCore{}
	project.AuthorId = authorId
	project.Json = emptyProjectJson2
	project.Name = "Untitled"

	projectId, createProjectErr := p.projectGateway.CreateProject(&project)
	if createProjectErr != nil {
		return nil, createProjectErr
	}

	projectPage := &models.ProjectPageCore{
		Title:       "Untitled",
		ProjectId:   projectId,
		Instruction: "",
		Notes:       "",
		Preview:     "",
		LinkScratch: viper.GetString("projectPage.scratchLink") + "?#" + projectId,
		IsShared:    false,
	}
	newProjectPage, err = p.projectPageGateway.CreateProjectPage(projectPage)
	return
}

func (p *ProjectPageUseCaseImpl) UpdateProjectPage(projectPage *models.ProjectPageCore, authorId string) (
	projectPageUpdated *models.ProjectPageCore,
	err error,
) {
	existing, err := p.projectPageGateway.GetProjectPageById(projectPage.ProjectPageId)
	if err != nil {
		return nil, err
	}
	// Права по фактической связке страницы с проектом в БД, а не по projectId из тела запроса
	// (иначе при рассинхроне URL/состояния UPDATE не попадает в строку, а API всё равно отвечает 200).
	_, err = p.projectGateway.GetProjectById(existing.ProjectId, authorId)
	if err != nil {
		return nil, err
	}
	projectPage.ProjectId = existing.ProjectId

	return p.projectPageGateway.UpdateProjectPage(projectPage)
}

func (p *ProjectPageUseCaseImpl) DeleteProjectPage(projectPageId string, authorId string) (err error) {
	projectPage, err := p.projectPageGateway.GetProjectPageById(projectPageId)
	if err != nil {
		return err
	}
	_, err = p.projectGateway.GetProjectById(projectPage.ProjectId, authorId)
	if err != nil {
		return err
	}
	// Метаданные страницы и scratch-проект — одна строка scratch_projects; достаточно одного soft delete.
	return p.projectGateway.DeleteProject(projectPage.ProjectId)
}

func (p *ProjectPageUseCaseImpl) GetAllProjectPageByUserId(authorId string, page, pageSize int) (
	projectPages []*models.ProjectPageCore,
	countRows int64,
	err error,
) {
	projects, countRows, getProjectsErr := p.projectGateway.GetProjectsByAuthorId(authorId, page, pageSize)
	if getProjectsErr != nil {
		return
	}
	for _, project := range projects {
		projectPage, errGetProjectPageById := p.projectPageGateway.GetProjectPageByProjectId(project.ID)
		if errGetProjectPageById != nil {
			return []*models.ProjectPageCore{}, 0, errGetProjectPageById
		}
		projectPages = append(projectPages, projectPage)
	}
	return
}

func (p *ProjectPageUseCaseImpl) GetProjectPageById(projectPageId string, authorId string) (
	projectPage *models.ProjectPageCore,
	err error,
) {
	projectPage, err = p.projectPageGateway.GetProjectPageById(projectPageId)
	if err != nil {
		return nil, err
	}

	// Проверяем, что проект страницы принадлежит пользователю.
	_, err = p.projectGateway.GetProjectById(projectPage.ProjectId, authorId)
	if err != nil {
		return nil, err
	}

	return projectPage, nil
}

func sb3DownloadFilename(title, fallbackPageID string) string {
	base := strings.TrimSpace(title)
	if base == "" {
		base = "project"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('_')
		case r == '.':
			b.WriteRune('.')
		default:
			if r < 128 {
				b.WriteRune('_')
			} else {
				b.WriteRune(r)
			}
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		short := fallbackPageID
		if len(short) > 12 {
			short = short[:12]
		}
		out = "project-" + short
	}
	const maxRunes = 120
	runes := []rune(out)
	if len(runes) > maxRunes {
		out = string(runes[:maxRunes])
	}
	return out + ".sb3"
}

func (p *ProjectPageUseCaseImpl) DownloadProjectSb3(projectPageId string, authorId string) (data []byte, filename string, err error) {
	core, err := p.GetProjectPageById(projectPageId, authorId)
	if err != nil {
		return nil, "", err
	}
	data, err = p.projectPageGateway.GetLatestSb3Archive(projectPageId)
	if err != nil {
		return nil, "", err
	}
	filename = sb3DownloadFilename(core.Title, core.ProjectPageId)
	return data, filename, nil
}
