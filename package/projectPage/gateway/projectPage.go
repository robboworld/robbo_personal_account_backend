package gateway

import (
	"crypto/sha256"
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := tx.Where("id = ? AND deleted_at IS NULL", rawID).First(&projectDB).Error; err != nil {
		return "", err
	}
	return rawID, nil
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

func (r *ProjectPageGatewayImpl) GetScratchProjectById(projectPageId string) (projectDB *models.ScratchProjectDB, err error) {
	var row models.ScratchProjectDB
	storageProjectID, resolveErr := r.resolveStorageProjectID(r.projectStorageDB, projectPageId)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil, projectPage.ErrPageNotFound
		}
		return nil, resolveErr
	}
	if err := r.projectStorageDB.Where("id = ? AND deleted_at IS NULL", storageProjectID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, projectPage.ErrPageNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *ProjectPageGatewayImpl) GetPublicProjectPages(page, pageSize int) (
	projectPages []*models.ProjectPageCore,
	countRows int64,
	err error,
) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	var rows []models.ScratchProjectDB
	err = r.projectStorageDB.
		Model(&models.ScratchProjectDB{}).
		Where("is_public = ? AND deleted_at IS NULL", true).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	r.projectStorageDB.Model(&models.ScratchProjectDB{}).
		Where("is_public = ? AND deleted_at IS NULL", true).
		Count(&countRows)
	for i := range rows {
		core := toProjectPageCore(&rows[i])
		projectPages = append(projectPages, core)
	}
	return projectPages, countRows, nil
}

// SaveSb3Archive stores a new .sb3 snapshot (or reuses an identical checksum).
func (r *ProjectPageGatewayImpl) SaveSb3Archive(projectPageId, userID string, archive []byte, saveSource string) error {
	if len(archive) == 0 {
		return projectPage.ErrBadRequest
	}
	hash := sha256.Sum256(archive)
	return r.projectStorageDB.Transaction(func(tx *gorm.DB) error {
		storageProjectID, err := r.resolveStorageProjectID(tx, projectPageId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return projectPage.ErrPageNotFound
			}
			return err
		}

		var existing models.ScratchProjectVersionDB
		dupErr := tx.Where("project_id = ? AND checksum_sha256 = ?", storageProjectID, hash[:]).
			First(&existing).Error
		if dupErr == nil {
			return tx.Model(&models.ScratchProjectDB{}).Where("id = ?", storageProjectID).
				Update("current_version_id", existing.ID).Error
		}
		if dupErr != nil && !errors.Is(dupErr, gorm.ErrRecordNotFound) {
			return dupErr
		}

		var project models.ScratchProjectDB
		if err := tx.Where("id = ? AND deleted_at IS NULL", storageProjectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return projectPage.ErrPageNotFound
			}
			return err
		}

		nextSeq := project.VersionCounter + 1
		if nextSeq < 1 {
			nextSeq = 1
		}
		src := saveSource
		ver := models.ScratchProjectVersionDB{
			ProjectID:       storageProjectID,
			VersionSeq:      nextSeq,
			Archive:         archive,
			SizeBytes:       int64(len(archive)),
			ChecksumSHA256:  hash[:],
			CreatedByUserID: userID,
			SaveSource:      &src,
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return tx.Model(&models.ScratchProjectDB{}).Where("id = ?", storageProjectID).Updates(map[string]interface{}{
			"version_counter":    nextSeq,
			"current_version_id": ver.ID,
		}).Error
	})
}

func (r *ProjectPageGatewayImpl) ListEnabledReactionTypes() ([]models.ReactionTypeHTTP, error) {
	var rows []models.ScratchReactionTypeDB
	err := r.projectStorageDB.
		Where("is_enabled = ?", true).
		Order("sort_order ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]models.ReactionTypeHTTP, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.ReactionTypeHTTP{
			Code:  row.Code,
			Emoji: row.Emoji,
		})
	}
	return out, nil
}

func (r *ProjectPageGatewayImpl) GetProjectReactionSummary(projectId, viewerUserId string) (*models.ProjectReactionsHTTP, error) {
	counts := make([]models.ReactionCountHTTP, 0)
	err := r.projectStorageDB.Raw(`
		SELECT t.code, t.emoji, COUNT(reaction.user_id) AS count
		FROM scratch_reaction_types t
		LEFT JOIN scratch_project_reactions reaction
		  ON reaction.reaction_code = t.code AND reaction.project_id = ?
		WHERE t.is_enabled = TRUE
		GROUP BY t.code, t.emoji, t.sort_order
		ORDER BY t.sort_order ASC
	`, projectId).Scan(&counts).Error
	if err != nil {
		return nil, err
	}

	var myReaction *string
	if viewerUserId != "" {
		var row models.ScratchProjectReactionDB
		err = r.projectStorageDB.
			Where("project_id = ? AND user_id = ?", projectId, viewerUserId).
			First(&row).Error
		switch {
		case err == nil:
			code := row.ReactionCode
			myReaction = &code
		case errors.Is(err, gorm.ErrRecordNotFound):
			myReaction = nil
		default:
			return nil, err
		}
	}

	return &models.ProjectReactionsHTTP{
		Counts:     counts,
		MyReaction: myReaction,
	}, nil
}

func (r *ProjectPageGatewayImpl) UpsertProjectReaction(projectId, userId, reactionCode string) error {
	reaction := models.ScratchProjectReactionDB{
		ProjectID:    projectId,
		UserID:       userId,
		ReactionCode: reactionCode,
	}
	return r.projectStorageDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "project_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"reaction_code": reactionCode,
			"updated_at":    gorm.Expr("now()"),
		}),
	}).Create(&reaction).Error
}

func (r *ProjectPageGatewayImpl) DeleteProjectReaction(projectId, userId string) error {
	return r.projectStorageDB.
		Where("project_id = ? AND user_id = ?", projectId, userId).
		Delete(&models.ScratchProjectReactionDB{}).Error
}
