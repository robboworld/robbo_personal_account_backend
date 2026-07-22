package usecase

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/access"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/playtoken"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type ProjectPageUseCaseImpl struct {
	projectPageGateway  projectPage.Gateway
	projectGateway      projects.Gateway
	notificationGateway notifications.Gateway
}

type ProjectPageUseCaseModule struct {
	fx.Out
	projectPage.UseCase
}

func SetupProjectPageUseCase(
	projectPageGateway projectPage.Gateway,
	projectGateway projects.Gateway,
	notificationGateway notifications.Gateway,
) ProjectPageUseCaseModule {
	return ProjectPageUseCaseModule{
		UseCase: &ProjectPageUseCaseImpl{
			projectPageGateway:  projectPageGateway,
			projectGateway:      projectGateway,
			notificationGateway: notificationGateway,
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
	if err == nil && newProjectPage != nil {
		newProjectPage.AuthorUserId = authorId
		newProjectPage.AuthorName = lookupAuthorName(authorId)
		newProjectPage.IsOwner = true
	}
	return
}

func (p *ProjectPageUseCaseImpl) enrichAndCheckRead(projectPageId, viewerId string) (
	core *models.ProjectPageCore,
	row *models.ScratchProjectDB,
	acc access.ProjectAccess,
	err error,
) {
	row, err = p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, nil, access.ProjectAccess{}, err
	}
	acc = access.Resolve(viewerId, row)
	if !acc.CanRead {
		return nil, nil, acc, auth.ErrNotAccess
	}
	core, err = p.projectPageGateway.GetProjectPageById(projectPageId)
	if err != nil {
		return nil, nil, acc, err
	}
	core.AuthorUserId = row.OwnerUserID
	core.AuthorName = lookupAuthorName(row.OwnerUserID)
	core.IsOwner = acc.IsOwner
	return core, row, acc, nil
}

func (p *ProjectPageUseCaseImpl) enrichPublicList(pages []*models.ProjectPageCore) {
	for _, core := range pages {
		if core == nil {
			continue
		}
		row, err := p.projectPageGateway.GetScratchProjectById(core.ProjectPageId)
		if err != nil {
			continue
		}
		core.AuthorUserId = row.OwnerUserID
		core.AuthorName = lookupAuthorName(row.OwnerUserID)
		core.IsOwner = false
	}
}

func (p *ProjectPageUseCaseImpl) UpdateProjectPage(projectPage *models.ProjectPageCore, authorId string) (
	projectPageUpdated *models.ProjectPageCore,
	err error,
) {
	existing, err := p.projectPageGateway.GetProjectPageById(projectPage.ProjectPageId)
	if err != nil {
		return nil, err
	}
	row, err := p.projectPageGateway.GetScratchProjectById(existing.ProjectPageId)
	if err != nil {
		return nil, err
	}
	if !access.Resolve(authorId, row).CanWrite {
		return nil, auth.ErrNotAccess
	}
	projectPage.ProjectId = existing.ProjectId

	updated, err := p.projectPageGateway.UpdateProjectPage(projectPage)
	if err != nil {
		return nil, err
	}
	updated.AuthorUserId = row.OwnerUserID
	updated.AuthorName = lookupAuthorName(row.OwnerUserID)
	updated.IsOwner = true
	return updated, nil
}

func (p *ProjectPageUseCaseImpl) DeleteProjectPage(projectPageId string, authorId string) (err error) {
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return err
	}
	if !access.Resolve(authorId, row).CanWrite {
		return auth.ErrNotAccess
	}
	return p.projectGateway.DeleteProject(row.ID)
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
		projectPage.AuthorUserId = authorId
		projectPage.AuthorName = lookupAuthorName(authorId)
		projectPage.IsOwner = true
		projectPages = append(projectPages, projectPage)
	}
	return
}

func (p *ProjectPageUseCaseImpl) GetProjectPageById(projectPageId string, viewerId string) (
	projectPage *models.ProjectPageCore,
	err error,
) {
	core, _, _, err := p.enrichAndCheckRead(projectPageId, viewerId)
	return core, err
}

func (p *ProjectPageUseCaseImpl) GetPublicProjectPages(page, pageSize int, landingFeaturedOnly bool) (
	projectPages []*models.ProjectPageCore,
	countRows int64,
	err error,
) {
	projectPages, countRows, err = p.projectPageGateway.GetPublicProjectPages(page, pageSize, landingFeaturedOnly)
	if err != nil {
		return nil, 0, err
	}
	p.enrichPublicList(projectPages)
	return projectPages, countRows, nil
}

func (p *ProjectPageUseCaseImpl) GetPreviewImage(projectPageId string, viewerId string) (data []byte, mime string, err error) {
	_, _, _, err = p.enrichAndCheckRead(projectPageId, viewerId)
	if err != nil {
		return nil, "", err
	}
	return p.projectPageGateway.GetPreviewImage(projectPageId)
}

func (p *ProjectPageUseCaseImpl) SavePreviewImage(projectPageId string, ownerId string, data []byte, mime string) error {
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return err
	}
	if !access.Resolve(ownerId, row).CanWrite {
		return auth.ErrNotAccess
	}
	return p.projectPageGateway.SavePreviewImage(projectPageId, data, mime)
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

func (p *ProjectPageUseCaseImpl) DownloadProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error) {
	core, err := p.GetProjectPageById(projectPageId, viewerId)
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

func (p *ProjectPageUseCaseImpl) PlayProjectSb3(projectPageId string, viewerId string) (data []byte, filename string, err error) {
	return p.DownloadProjectSb3(projectPageId, viewerId)
}

func (p *ProjectPageUseCaseImpl) PlayProjectSb3ByToken(projectPageId string, token string) (data []byte, filename string, err error) {
	claims, err := playtoken.Parse(token)
	if err != nil {
		return nil, "", auth.ErrInvalidAccessToken
	}
	if claims.ProjectPageID != projectPageId {
		return nil, "", auth.ErrInvalidAccessToken
	}
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, "", err
	}
	if !row.IsPublic && claims.ProjectID != row.ID {
		return nil, "", auth.ErrInvalidAccessToken
	}
	data, err = p.projectPageGateway.GetLatestSb3Archive(projectPageId)
	if err != nil {
		return nil, "", err
	}
	filename = sb3DownloadFilename(row.Title, projectPageId)
	return data, filename, nil
}

func (p *ProjectPageUseCaseImpl) ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error) {
	return p.projectPageGateway.ListEnabledReactionTypes()
}

func (p *ProjectPageUseCaseImpl) GetProjectReactions(
	projectPageId string,
	viewerId string,
) (*models.ProjectReactionsHTTP, error) {
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, err
	}
	if !row.IsPublic {
		return nil, auth.ErrNotAccess
	}
	return p.projectPageGateway.GetProjectReactionSummary(projectPageId, strings.TrimSpace(viewerId))
}

func (p *ProjectPageUseCaseImpl) PutProjectReaction(
	projectPageId string,
	userId string,
	reactionCode string,
) (*models.ProjectReactionsHTTP, error) {
	userId = strings.TrimSpace(userId)
	reactionCode = strings.TrimSpace(reactionCode)
	if userId == "" || reactionCode == "" {
		return nil, projectPage.ErrBadRequest
	}

	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, err
	}
	if !row.IsPublic {
		return nil, auth.ErrNotAccess
	}

	types, err := p.projectPageGateway.ListEnabledReactionTypes()
	if err != nil {
		return nil, err
	}
	var emoji string
	for _, reactionType := range types {
		if reactionType.Code == reactionCode {
			emoji = reactionType.Emoji
			break
		}
	}
	if emoji == "" {
		return nil, projectPage.ErrBadRequest
	}

	if err := p.projectPageGateway.UpsertProjectReaction(projectPageId, userId, reactionCode); err != nil {
		return nil, err
	}

	if strings.TrimSpace(row.OwnerUserID) != userId {
		actionURL := "/projects/" + projectPageId
		dedupeKey := "project_reaction:" + projectPageId + ":" + userId
		if err := p.notificationGateway.CreateOrUpdateByDedupe(&models.UserNotificationDB{
			RecipientUserID: strings.TrimSpace(row.OwnerUserID),
			Title:           "Новая реакция на проект",
			Body:            fmt.Sprintf("На ваш проект «%s» поставили %s", row.Title, emoji),
			Kind:            "project_reaction",
			Severity:        "INFO",
			Source:          "system",
			ActionURL:       &actionURL,
			DedupeKey:       &dedupeKey,
		}); err != nil {
			return nil, err
		}
	}

	return p.projectPageGateway.GetProjectReactionSummary(projectPageId, userId)
}

func (p *ProjectPageUseCaseImpl) DeleteProjectReaction(
	projectPageId string,
	userId string,
) (*models.ProjectReactionsHTTP, error) {
	userId = strings.TrimSpace(userId)
	if userId == "" {
		return nil, projectPage.ErrBadRequest
	}
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, err
	}
	if !row.IsPublic {
		return nil, auth.ErrNotAccess
	}
	if err := p.projectPageGateway.DeleteProjectReaction(projectPageId, userId); err != nil {
		return nil, err
	}
	return p.projectPageGateway.GetProjectReactionSummary(projectPageId, userId)
}

func (p *ProjectPageUseCaseImpl) GetProjectJSONByToken(projectId string, token string) (string, error) {
	claims, err := playtoken.Parse(token)
	if err != nil {
		return "", auth.ErrInvalidAccessToken
	}
	if claims.ProjectID != projectId {
		return "", auth.ErrInvalidAccessToken
	}
	row, err := p.projectPageGateway.GetScratchProjectById(projectId)
	if err != nil {
		return "", err
	}
	if row.ID != projectId {
		return "", auth.ErrInvalidAccessToken
	}
	vmJSON := row.ScratchVMJSON
	if vmJSON == "" {
		vmJSON = "{}"
	}
	return vmJSON, nil
}

func (p *ProjectPageUseCaseImpl) IssuePlayToken(projectPageId string, viewerId string, backendBaseURL string) (*projectPage.PlayTokenHTTP, error) {
	core, row, acc, err := p.enrichAndCheckRead(projectPageId, viewerId)
	if err != nil {
		return nil, err
	}
	_ = acc
	token, expiresAt, err := playtoken.Issue(core.ProjectPageId, core.ProjectId)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(backendBaseURL, "/")
	q := url.QueryEscape(token)
	playURL := ""
	if _, errSb3 := p.projectPageGateway.GetLatestSb3Archive(projectPageId); errSb3 == nil {
		playURL = fmt.Sprintf("%s/projectPage/%s/play?token=%s", base, url.PathEscape(projectPageId), q)
	}
	jsonURL := fmt.Sprintf("%s/project/%s?token=%s", base, url.PathEscape(row.ID), q)
	return &projectPage.PlayTokenHTTP{
		PlayURL:   playURL,
		JSONURL:   jsonURL,
		ExpiresAt: expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (p *ProjectPageUseCaseImpl) UploadProjectSb3(projectPageId string, ownerId string, data []byte) error {
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return err
	}
	if !access.Resolve(ownerId, row).CanWrite {
		return auth.ErrNotAccess
	}
	if err := p.projectPageGateway.SaveSb3Archive(projectPageId, ownerId, data, "lk.upload"); err != nil {
		return err
	}
	if vmJSON, extractErr := extractProjectJSONFromSb3(data); extractErr == nil && strings.TrimSpace(vmJSON) != "" {
		updateErr := p.projectGateway.UpdateProject(&models.ProjectCore{
			ID:   row.ID,
			Json: vmJSON,
		})
		if updateErr != nil {
			return updateErr
		}
	}
	return nil
}
