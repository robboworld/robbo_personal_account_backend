package gateway

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationGateway struct {
	db *gorm.DB
}

type Module struct {
	fx.Out
	notifications.Gateway
}

func SetupNotificationGateway(postgresClient db_client.PostgresClient) Module {
	_ = postgresClient
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		panic("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return Module{Gateway: &NotificationGateway{db: db}}
}

func (g *NotificationGateway) CreateOrUpdateByDedupe(notification *models.UserNotificationDB) error {
	return g.db.Exec(`
		INSERT INTO user_notifications (
			recipient_user_id, title, body, kind, severity, source, action_url, dedupe_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL
		DO UPDATE SET
			recipient_user_id = EXCLUDED.recipient_user_id,
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			kind = EXCLUDED.kind,
			severity = EXCLUDED.severity,
			source = EXCLUDED.source,
			action_url = EXCLUDED.action_url,
			read_at = NULL,
			created_at = now()
	`,
		notification.RecipientUserID,
		notification.Title,
		notification.Body,
		notification.Kind,
		notification.Severity,
		notification.Source,
		notification.ActionURL,
		notification.DedupeKey,
	).Error
}

func (g *NotificationGateway) CreatePersonal(notification *models.UserNotificationDB) error {
	return g.db.Create(notification).Error
}

func (g *NotificationGateway) ListFeed(userID string, limit int) ([]models.NotificationFeedItemHTTP, error) {
	var items []models.NotificationFeedItemHTTP
	err := g.db.Raw(`
		SELECT
			'personal' AS feed_kind,
			n.id::text AS id,
			n.title,
			n.body,
			n.kind,
			n.severity,
			n.source,
			n.read_at,
			n.created_at,
			n.action_url
		FROM user_notifications n
		WHERE n.recipient_user_id = ?
		UNION ALL
		SELECT
			'announcement' AS feed_kind,
			a.id::text AS id,
			a.title,
			a.body,
			'announcement' AS kind,
			'INFO' AS severity,
			'system' AS source,
			ar.read_at,
			a.created_at,
			NULL::text AS action_url
		FROM system_announcements a
		LEFT JOIN announcement_reads ar
			ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE (a.expires_at IS NULL OR a.expires_at > now())
		  AND COALESCE((a.audience_json::jsonb ->> 'all')::boolean, FALSE) = TRUE
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, userID, limit).Scan(&items).Error
	return items, err
}

func (g *NotificationGateway) CountUnread(userID string) (int64, error) {
	var result struct {
		Count int64 `gorm:"column:count"`
	}
	err := g.db.Raw(`
		SELECT (
			(SELECT COUNT(*) FROM user_notifications
			 WHERE recipient_user_id = ? AND read_at IS NULL)
			+
			(SELECT COUNT(*)
			 FROM system_announcements a
			 LEFT JOIN announcement_reads ar
			   ON ar.announcement_id = a.id AND ar.user_id = ?
			 WHERE ar.announcement_id IS NULL
			   AND (a.expires_at IS NULL OR a.expires_at > now())
			   AND COALESCE((a.audience_json::jsonb ->> 'all')::boolean, FALSE) = TRUE)
		) AS count
	`, userID, userID).Scan(&result).Error
	return result.Count, err
}

func (g *NotificationGateway) MarkPersonalRead(userID, notificationID string) error {
	return g.db.Model(&models.UserNotificationDB{}).
		Where("id = ? AND recipient_user_id = ?", notificationID, userID).
		Update("read_at", time.Now()).Error
}

func (g *NotificationGateway) MarkAnnouncementRead(userID, announcementID string) error {
	row := models.AnnouncementReadDB{
		AnnouncementID: announcementID,
		UserID:         userID,
		ReadAt:         time.Now(),
	}
	return g.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (g *NotificationGateway) MarkAllRead(userID string) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.UserNotificationDB{}).
			Where("recipient_user_id = ? AND read_at IS NULL", userID).
			Update("read_at", time.Now()).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO announcement_reads (announcement_id, user_id, read_at)
			SELECT a.id, ?, now()
			FROM system_announcements a
			WHERE (a.expires_at IS NULL OR a.expires_at > now())
			  AND COALESCE((a.audience_json::jsonb ->> 'all')::boolean, FALSE) = TRUE
			ON CONFLICT (announcement_id, user_id) DO NOTHING
		`, userID).Error
	})
}

func (g *NotificationGateway) CreateAnnouncement(announcement *models.SystemAnnouncementDB) error {
	return g.db.Create(announcement).Error
}
