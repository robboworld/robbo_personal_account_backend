package gateway

import (
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// BansGatewayImpl stores ban history in Licensing Postgres.
type BansGatewayImpl struct {
	db *gorm.DB
}

// BansGatewayModule is the FX output for ban persistence.
type BansGatewayModule struct {
	fx.Out
	moderation.Gateway
}

// SetupBansGateway opens Licensing Postgres (same DSN as licensing gateway).
func SetupBansGateway(_ db_client.PostgresClient) BansGatewayModule {
	dsn := viper.GetString("licensingPostgres.postgresDsn")
	if dsn == "" {
		panic("licensingPostgres.postgresDsn (or env LICENSING_POSTGRES_DSN) is required for moderation")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return BansGatewayModule{
		Gateway: &BansGatewayImpl{db: db},
	}
}

func (g *BansGatewayImpl) CreateBan(ban *models.UserBanCore) (*models.UserBanCore, error) {
	now := time.Now().UTC()
	row := models.UserBanDB{
		LmsUserID:         ban.LmsUserID,
		Reason:            ban.Reason,
		BannedByLmsUserID: ban.BannedByLmsUserID,
		BannedAt:          now,
		ExpiresAt:         ban.ExpiresAt,
		RevokedSessions:   ban.RevokedSessions,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *BansGatewayImpl) GetActiveBan(lmsUserID string) (*models.UserBanCore, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	var row models.UserBanDB
	err := g.db.Where(
		"lms_user_id = ? AND unbanned_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
		lmsUserID, now,
	).Order("banned_at DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// GetOpenBan returns the latest ban row that has not been closed yet,
// including temporary bans that have already expired (for auto-unban).
func (g *BansGatewayImpl) GetOpenBan(lmsUserID string) (*models.UserBanCore, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.UserBanDB
	err := g.db.Where("lms_user_id = ? AND unbanned_at IS NULL", lmsUserID).
		Order("banned_at DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *BansGatewayImpl) ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return nil, nil
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows []models.UserBanDB
	err := g.db.Where("lms_user_id = ?", lmsUserID).
		Order("banned_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*models.UserBanCore, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToCore())
	}
	return out, nil
}

func (g *BansGatewayImpl) CloseBan(banID, unbannedBy, unbanReason string) (*models.UserBanCore, error) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"unbanned_at":            now,
		"unbanned_by_lms_user_id": unbannedBy,
	}
	if strings.TrimSpace(unbanReason) != "" {
		updates["unban_reason"] = strings.TrimSpace(unbanReason)
	}
	res := g.db.Model(&models.UserBanDB{}).
		Where("id = ? AND unbanned_at IS NULL", banID).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.UserBanDB
	if err := g.db.Where("id = ?", banID).First(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *BansGatewayImpl) ListExpiredActiveBans(now time.Time, limit int) ([]*models.UserBanCore, error) {
	if limit < 1 {
		limit = 50
	}
	var rows []models.UserBanDB
	err := g.db.Where(
		"unbanned_at IS NULL AND expires_at IS NOT NULL AND expires_at <= ?",
		now,
	).Order("expires_at ASC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*models.UserBanCore, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToCore())
	}
	return out, nil
}
