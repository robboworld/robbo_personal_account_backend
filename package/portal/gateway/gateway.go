package gateway

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type Gateway interface {
	UpsertUserLinkByOIDC(sub, email, name, edxUserID string) (*models.RobboPortalUserLinkDB, error)
	FindUserLinkByEmail(email string) (*models.RobboPortalUserLinkDB, error)
	FindUserLinkByLegacyOrEdx(id string) (*models.RobboPortalUserLinkDB, error)
	FindPrimaryRoleByUserLinkID(userLinkID int64) (models.Role, bool, error)
	EnsureDefaultRole(userLinkID int64, roleCode string) error
	EnqueueOutbox(idempotencyKey, eventType string, payload interface{}) (created bool, err error)
	FetchPendingOutbox(limit int) ([]models.RobboPortalIntegrationOutboxDB, error)
	MarkOutboxDone(id int64) error
	MarkOutboxFailed(id int64, errMsg string) error
	CreateNotification(n *models.RobboPortalNotificationDB) (duplicate bool, err error)
}

type GatewayImpl struct {
	db *gorm.DB
}

type GatewayModule struct {
	fx.Out
	Gateway
}

func SetupPortalGateway() GatewayModule {
	if !viper.GetBool("legacyPostgres.enabled") {
		return GatewayModule{Gateway: NoopGateway{}}
	}
	dsn := viper.GetString("postgres.postgresDsn")
	if dsn == "" {
		panic("postgres.postgresDsn required for portal gateway when legacyPostgres.enabled is true")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return GatewayModule{Gateway: &GatewayImpl{db: db}}
}

func (g *GatewayImpl) UpsertUserLinkByOIDC(sub, email, name, edxUserID string) (*models.RobboPortalUserLinkDB, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var row models.RobboPortalUserLinkDB
	err := g.db.Where("lower(email) = ?", email).First(&row).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		subCopy := sub
		row = models.RobboPortalUserLinkDB{
			Email:       email,
			OIDCSub:     &subCopy,
			LastLoginAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if name != "" {
			row.DisplayName = &name
		}
		if edxUserID != "" {
			row.EdxUserID = &edxUserID
		}
		return &row, g.db.Create(&row).Error
	}
	if err != nil {
		return nil, err
	}
	row.LastLoginAt = &now
	row.UpdatedAt = now
	if sub != "" {
		row.OIDCSub = &sub
	}
	if edxUserID != "" {
		row.EdxUserID = &edxUserID
	}
	// Do not overwrite display_name on login — profile edits store FIO there.
	return &row, g.db.Save(&row).Error
}

func (g *GatewayImpl) FindUserLinkByEmail(email string) (*models.RobboPortalUserLinkDB, error) {
	var row models.RobboPortalUserLinkDB
	err := g.db.Where("lower(email) = ?", strings.TrimSpace(strings.ToLower(email))).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (g *GatewayImpl) FindPrimaryRoleByUserLinkID(userLinkID int64) (models.Role, bool, error) {
	var row models.RobboPortalRoleDB
	err := g.db.Where("user_link_id = ?", userLinkID).Order("id ASC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Student, false, nil
	}
	if err != nil {
		return models.Student, false, err
	}
	return roleCodeToModel(row.RoleCode), true, nil
}

func (g *GatewayImpl) EnsureDefaultRole(userLinkID int64, roleCode string) error {
	if roleCode == "" {
		roleCode = "student"
	}
	var existing models.RobboPortalRoleDB
	err := g.db.Where("user_link_id = ? AND role_code = ?", userLinkID, roleCode).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return g.db.Create(&models.RobboPortalRoleDB{
		UserLinkID: userLinkID,
		RoleCode:   roleCode,
		CreatedAt:  time.Now(),
	}).Error
}

func roleCodeToModel(code string) models.Role {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "teacher":
		return models.Teacher
	case "parent":
		return models.Parent
	case "free_listener":
		return models.FreeListener
	case "unit_admin":
		return models.UnitAdmin
	case "super_admin":
		return models.SuperAdmin
	default:
		return models.Student
	}
}

func (g *GatewayImpl) FindUserLinkByLegacyOrEdx(id string) (*models.RobboPortalUserLinkDB, error) {
	var row models.RobboPortalUserLinkDB
	err := g.db.Where("legacy_lk_user_id = ? OR edx_user_id = ?", id, id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (g *GatewayImpl) EnqueueOutbox(idempotencyKey, eventType string, payload interface{}) (bool, error) {
	var existing models.RobboPortalIntegrationOutboxDB
	if err := g.db.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
		return false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	row := models.RobboPortalIntegrationOutboxDB{
		IdempotencyKey: idempotencyKey,
		EventType:      eventType,
		Payload:        string(raw),
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
	return true, g.db.Create(&row).Error
}

func (g *GatewayImpl) FetchPendingOutbox(limit int) ([]models.RobboPortalIntegrationOutboxDB, error) {
	var rows []models.RobboPortalIntegrationOutboxDB
	q := g.db.Where("status = ?", "pending").Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return rows, q.Find(&rows).Error
}

func (g *GatewayImpl) MarkOutboxDone(id int64) error {
	now := time.Now()
	return g.db.Model(&models.RobboPortalIntegrationOutboxDB{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": "done", "processed_at": now}).Error
}

func (g *GatewayImpl) MarkOutboxFailed(id int64, errMsg string) error {
	var row models.RobboPortalIntegrationOutboxDB
	if err := g.db.First(&row, id).Error; err != nil {
		return err
	}
	row.Status = "failed"
	row.Attempts++
	row.LastError = &errMsg
	return g.db.Save(&row).Error
}

func (g *GatewayImpl) CreateNotification(n *models.RobboPortalNotificationDB) (bool, error) {
	if n.DedupeKey != nil && n.RecipientEmail != nil {
		var dup models.RobboPortalNotificationDB
		err := g.db.Where("recipient_email = ? AND dedupe_key = ?", *n.RecipientEmail, *n.DedupeKey).First(&dup).Error
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
	}
	n.CreatedAt = time.Now()
	return false, g.db.Create(n).Error
}
