package gateway

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type ProjectsGatewayImpl struct {
	projectStorageDB *gorm.DB
}

type ProjectsGatewayModule struct {
	fx.Out
	projects.Gateway
}

func SetupProjectsGateway(postgresClient db_client.PostgresClient) ProjectsGatewayModule {
	_ = postgresClient
	projectDSN := viper.GetString("projectsPostgres.postgresDsn")
	if projectDSN == "" {
		panic("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required; robbo_db fallback is disabled")
	}
	projectStorageDB, err := db_client.OpenByDSN(projectDSN)
	if err != nil {
		panic(err)
	}

	return ProjectsGatewayModule{
		Gateway: &ProjectsGatewayImpl{projectStorageDB: projectStorageDB},
	}
}

func (r *ProjectsGatewayImpl) resolveStorageProjectID(tx *gorm.DB, rawID string) (string, error) {
	var projectDB models.ScratchProjectDB
	if err := tx.Where("id = ? AND deleted_at IS NULL", rawID).First(&projectDB).Error; err == nil {
		return rawID, nil
	}

	var legacyMap models.ScratchProjectLegacyMapDB
	if err := tx.
		Where("legacy_project_id = ? OR legacy_project_page_id = ?", rawID, rawID).
		First(&legacyMap).Error; err != nil {
		return "", err
	}
	return legacyMap.StorageProjectID, nil
}

func (r *ProjectsGatewayImpl) CreateProject(project *models.ProjectCore) (id string, err error) {
	projectDB := models.ScratchProjectDB{
		OwnerUserID: project.AuthorId,
		Title:       project.Name,
	}
	if projectDB.Title == "" {
		projectDB.Title = "Untitled"
	}
	if project.Json != "" {
		projectDB.ScratchVMJSON = project.Json
	} else if projectDB.ScratchVMJSON == "" {
		projectDB.ScratchVMJSON = "{}"
	}

	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		err = tx.Create(&projectDB).Error
		return
	})
	if err != nil {
		return "", err
	}

	id = projectDB.ID
	return
}

func (r *ProjectsGatewayImpl) GetProjectById(projectId, userId string) (project *models.ProjectCore, err error) {
	var projectDB models.ScratchProjectDB
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		storageProjectID, resolveErr := r.resolveStorageProjectID(tx, projectId)
		if resolveErr != nil {
			return resolveErr
		}
		if err = tx.Where("id = ? AND deleted_at IS NULL", storageProjectID).First(&projectDB).Error; err != nil {
			return
		}
		return
	})
	if err != nil {
		return nil, err
	}

	vmJSON := projectDB.ScratchVMJSON
	if vmJSON == "" {
		vmJSON = "{}"
	}
	project = &models.ProjectCore{
		ID:       projectDB.ID,
		Name:     projectDB.Title,
		AuthorId: projectDB.OwnerUserID,
		Json:     vmJSON,
	}
	if project.AuthorId == userId {
		return
	} else {
		return nil, auth.ErrNotAccess
	}
}

func (r *ProjectsGatewayImpl) GetProjectsByAuthorId(authorId string, page, pageSize int) (
	projects []*models.ProjectCore,
	countRows int64,
	err error,
) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var projectsDB []*models.ScratchProjectDB
	offset := (page - 1) * pageSize
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		// Корректный порядок: фильтр → сортировка → limit/offset (иначе возможны некорректные выборки).
		if err = tx.
			Model(&models.ScratchProjectDB{}).
			Where("owner_user_id = ? AND deleted_at IS NULL", authorId).
			Order("updated_at DESC").
			Limit(pageSize).
			Offset(offset).
			Find(&projectsDB).Error; err != nil {
			return
		}
		tx.Model(&models.ScratchProjectDB{}).
			Where("owner_user_id = ? AND deleted_at IS NULL", authorId).
			Count(&countRows)
		return
	})
	if err != nil {
		return nil, 0, err
	}

	for _, projectDB := range projectsDB {
		vmJSON := projectDB.ScratchVMJSON
		if vmJSON == "" {
			vmJSON = "{}"
		}
		projects = append(projects, &models.ProjectCore{
			ID:       projectDB.ID,
			Name:     projectDB.Title,
			AuthorId: projectDB.OwnerUserID,
			Json:     vmJSON,
		})
	}
	return
}

func (r *ProjectsGatewayImpl) DeleteProject(projectId string) (err error) {
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		storageProjectID, resolveErr := r.resolveStorageProjectID(tx, projectId)
		if resolveErr != nil {
			return resolveErr
		}
		err = tx.Where("id = ?", storageProjectID).Delete(&models.ScratchProjectDB{}).Error
		return
	})
	return
}

func (r *ProjectsGatewayImpl) UpdateProject(project *models.ProjectCore) (err error) {
	updates := map[string]interface{}{}
	if project.Name != "" {
		updates["title"] = project.Name
	}
	if project.Json != "" {
		updates["scratch_vm_json"] = project.Json
	}
	if len(updates) == 0 {
		return nil
	}
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		storageProjectID, resolveErr := r.resolveStorageProjectID(tx, project.ID)
		if resolveErr != nil {
			return resolveErr
		}
		err = tx.Model(&models.ScratchProjectDB{}).Where("id = ? AND deleted_at IS NULL", storageProjectID).Updates(updates).Error
		return
	})
	return
}
