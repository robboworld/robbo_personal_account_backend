package gateway

import (
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type ProjectPageGatewayImpl struct {
	projectStorageDB *gorm.DB
}

type ProjectPageGatewayModule struct {
	fx.Out
	projectPage.Gateway
}

func SetupProjectPageGateway(postgresClient db_client.PostgresClient) ProjectPageGatewayModule {
	_ = postgresClient
	projectDSN := viper.GetString("projectsPostgres.postgresDsn")
	if projectDSN == "" {
		panic("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required; robbo_db fallback is disabled")
	}
	projectStorageDB, err := db_client.OpenByDSN(projectDSN)
	if err != nil {
		panic(err)
	}

	return ProjectPageGatewayModule{
		Gateway: &ProjectPageGatewayImpl{projectStorageDB: projectStorageDB},
	}
}

func (r *ProjectPageGatewayImpl) CreateProjectPage(page *models.ProjectPageCore) (newProjectPage *models.ProjectPageCore, err error) {
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		updates := map[string]interface{}{
			"title":       page.Title,
			"instruction": page.Instruction,
			"note":        page.Notes,
			"is_public":   page.IsShared,
		}
		res := tx.Model(&models.ScratchProjectDB{}).
			Where("id = ? AND deleted_at IS NULL", page.ProjectId).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return projectPage.ErrPageNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetProjectPageByProjectId(page.ProjectId)
}

func toProjectPageCore(projectDB *models.ScratchProjectDB) *models.ProjectPageCore {
	scratchLink := viper.GetString("projectPage.scratchLink") + "?#" + projectDB.ID
	return &models.ProjectPageCore{
		ProjectPageId: projectDB.ID,
		LastModified:  projectDB.UpdatedAt.String(),
		Title:         projectDB.Title,
		ProjectId:     projectDB.ID,
		Instruction:   projectDB.Instruction,
		Notes:         projectDB.Note,
		Preview:       "",
		LinkScratch:   scratchLink,
		IsShared:      projectDB.IsPublic,
	}
}

func (r *ProjectPageGatewayImpl) resolveStorageProjectID(tx *gorm.DB, rawID string) (string, error) {
	var projectDB models.ScratchProjectDB
	if err := tx.Where("id = ? AND deleted_at IS NULL", rawID).First(&projectDB).Error; err == nil {
		return rawID, nil
	}

	var legacyMap models.ScratchProjectLegacyMapDB
	if err := tx.
		Where("legacy_project_page_id = ? OR legacy_project_id = ?", rawID, rawID).
		First(&legacyMap).Error; err != nil {
		return "", err
	}
	return legacyMap.StorageProjectID, nil
}

func (r *ProjectPageGatewayImpl) GetProjectPageById(projectPageId string) (projectPageCore *models.ProjectPageCore, err error) {
	var projectDB models.ScratchProjectDB
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		storageProjectID, resolveErr := r.resolveStorageProjectID(tx, projectPageId)
		if resolveErr != nil {
			return resolveErr
		}
		if err = tx.Where("id = ? AND deleted_at IS NULL", storageProjectID).First(&projectDB).Error; err != nil {
			return
		}
		return
	})
	projectPageCore = toProjectPageCore(&projectDB)
	return
}

func (r *ProjectPageGatewayImpl) GetProjectPageByProjectId(projectId string) (projectPageCore *models.ProjectPageCore, err error) {
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
	projectPageCore = toProjectPageCore(&projectDB)
	return
}

func (r *ProjectPageGatewayImpl) DeleteProjectPage(projectId string) (err error) {
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

func (r *ProjectPageGatewayImpl) UpdateProjectPage(core *models.ProjectPageCore) (projectPageUpdated *models.ProjectPageCore, err error) {
	err = r.projectStorageDB.Transaction(func(tx *gorm.DB) (err error) {
		updates := map[string]interface{}{
			"instruction": core.Instruction,
			"note":        core.Notes,
			"title":       core.Title,
			"is_public":   core.IsShared,
		}
		targetID := core.ProjectPageId
		if targetID == "" {
			targetID = core.ProjectId
		}
		storageProjectID, resolveErr := r.resolveStorageProjectID(tx, targetID)
		if resolveErr != nil {
			return resolveErr
		}
		res := tx.Model(&models.ScratchProjectDB{}).
			Where("id = ? AND deleted_at IS NULL", storageProjectID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return projectPage.ErrPageNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	targetID := core.ProjectPageId
	if targetID == "" {
		targetID = core.ProjectId
	}
	return r.GetProjectPageById(targetID)
}

// GetLatestSb3Archive loads the archived .sb3 (BYTEA): current_version_id if valid, otherwise latest version_seq.
func (r *ProjectPageGatewayImpl) GetLatestSb3Archive(projectPageId string) (archive []byte, err error) {
	storageProjectID, resolveErr := r.resolveStorageProjectID(r.projectStorageDB, projectPageId)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil, projectPage.ErrPageNotFound
		}
		return nil, resolveErr
	}

	var projectDB models.ScratchProjectDB
	if err := r.projectStorageDB.Where("id = ? AND deleted_at IS NULL", storageProjectID).First(&projectDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, projectPage.ErrPageNotFound
		}
		return nil, err
	}

	var ver models.ScratchProjectVersionDB

	if projectDB.CurrentVersionID != nil && *projectDB.CurrentVersionID != "" {
		errCur := r.projectStorageDB.Model(&models.ScratchProjectVersionDB{}).Select("archive").
			Where("id = ? AND project_id = ?", *projectDB.CurrentVersionID, storageProjectID).
			First(&ver).Error
		if errCur == nil && len(ver.Archive) > 0 {
			return ver.Archive, nil
		}
		if errCur != nil && !errors.Is(errCur, gorm.ErrRecordNotFound) {
			return nil, errCur
		}
	}

	err = r.projectStorageDB.Model(&models.ScratchProjectVersionDB{}).Select("archive").
		Where("project_id = ?", storageProjectID).
		Order("version_seq DESC").
		First(&ver).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, projectPage.ErrSb3ArchiveNotFound
		}
		return nil, err
	}
	if len(ver.Archive) == 0 {
		return nil, projectPage.ErrSb3ArchiveNotFound
	}
	return ver.Archive, nil
}
