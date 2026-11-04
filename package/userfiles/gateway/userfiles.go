package gateway

import (
	"errors"
	"fmt"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type UserFilesGatewayImpl struct {
	db *gorm.DB
}

func SetupUserFilesGateway() (*UserFilesGatewayImpl, error) {
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		return nil, errors.New("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("SetupUserFilesGateway: %w", err)
	}
	return &UserFilesGatewayImpl{db: db}, nil
}

func NewUserFilesGateway(db *gorm.DB) *UserFilesGatewayImpl {
	return &UserFilesGatewayImpl{db: db}
}

func live(db *gorm.DB, ownerUserID string) *gorm.DB {
	return db.Model(&models.UserFileDB{}).Where("owner_user_id = ? AND deleted_at IS NULL", ownerUserID)
}

func (g *UserFilesGatewayImpl) GetTotalStorageBytesForOwner(ownerUserID string) (int64, error) {
	var total int64
	err := live(g.db, ownerUserID).Select("COALESCE(SUM(size_bytes), 0)").Scan(&total).Error
	return total, err
}

func (g *UserFilesGatewayImpl) CountForOwner(ownerUserID string) (int64, error) {
	var n int64
	err := live(g.db, ownerUserID).Count(&n).Error
	return n, err
}

func (g *UserFilesGatewayImpl) CountCreatedSince(ownerUserID string, t time.Time) (int64, error) {
	var n int64
	err := g.db.Model(&models.UserFileDB{}).
		Where("owner_user_id = ? AND created_at >= ?", ownerUserID, t).
		Count(&n).Error
	return n, err
}

func (g *UserFilesGatewayImpl) ListByOwner(ownerUserID string) ([]*models.UserFileDB, error) {
	var rows []*models.UserFileDB
	err := live(g.db, ownerUserID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (g *UserFilesGatewayImpl) GetByOwnerAndID(ownerUserID, id string) (*models.UserFileDB, error) {
	var row models.UserFileDB
	err := live(g.db, ownerUserID).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, userfiles.ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateLocked runs check and the insert under one advisory lock per owner. check reads the
// usage through other connections; that is safe because the previous holder committed
// before it released the lock.
func (g *UserFilesGatewayImpl) CreateLocked(row *models.UserFileDB, check func() error) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "storage:"+row.OwnerUserID).Error; err != nil {
			return err
		}
		if err := check(); err != nil {
			return err
		}
		return tx.Create(row).Error
	})
}

func (g *UserFilesGatewayImpl) SoftDelete(ownerUserID, id string) error {
	res := live(g.db, ownerUserID).Where("id = ?", id).Update("deleted_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return userfiles.ErrFileNotFound
	}
	return nil
}

func (g *UserFilesGatewayImpl) MarkScanned(id string, at time.Time) error {
	return g.db.Model(&models.UserFileDB{}).Where("id = ?", id).Update("scanned_at", at).Error
}

func (g *UserFilesGatewayImpl) ListDeletedBefore(t time.Time, limit int) ([]string, error) {
	var ids []string
	err := g.db.Model(&models.UserFileDB{}).
		Where("deleted_at IS NOT NULL AND deleted_at < ?", t).
		Order("deleted_at").Limit(limit).
		Pluck("id", &ids).Error
	return ids, err
}

func (g *UserFilesGatewayImpl) HardDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return g.db.Where("id IN ? AND deleted_at IS NOT NULL", ids).Delete(&models.UserFileDB{}).Error
}
