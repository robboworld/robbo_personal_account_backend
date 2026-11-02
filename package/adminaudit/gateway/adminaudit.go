package gateway

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type AdminAuditGatewayImpl struct {
	db *gorm.DB
}

func SetupAdminAuditGateway() (*AdminAuditGatewayImpl, error) {
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		return nil, errors.New("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("SetupAdminAuditGateway: %w", err)
	}
	return &AdminAuditGatewayImpl{db: db}, nil
}

func (g *AdminAuditGatewayImpl) Write(entry *models.AdminContentAuditDB) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	return g.db.Create(entry).Error
}

func (g *AdminAuditGatewayImpl) ListForTarget(targetUserID string, limit int) ([]models.AdminContentAuditDB, error) {
	var rows []models.AdminContentAuditDB
	err := g.db.Where("target_user_id = ?", targetUserID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
