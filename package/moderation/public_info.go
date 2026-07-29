package moderation

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// PublicBanInfo is safe to show on the login screen to a banned user.
type PublicBanInfo struct {
	Reason      string     `json:"reason"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	IsPermanent bool       `json:"isPermanent"`
}

var (
	publicBanDBOnce sync.Once
	publicBanDB     *gorm.DB
)

func publicBanDBConn() *gorm.DB {
	publicBanDBOnce.Do(func() {
		dsn := viper.GetString("licensingPostgres.postgresDsn")
		if dsn == "" {
			return
		}
		db, err := db_client.OpenByDSN(dsn)
		if err != nil {
			log.Printf("moderation: public ban db: %v", err)
			return
		}
		publicBanDB = db
	})
	return publicBanDB
}

// LookupPublicBanInfo returns active ban details for an LMS user id, or nil.
func LookupPublicBanInfo(lmsUserID string) *PublicBanInfo {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return nil
	}
	db := publicBanDBConn()
	if db == nil {
		return nil
	}
	now := time.Now().UTC()
	var row models.UserBanDB
	err := db.Where(
		"lms_user_id = ? AND unbanned_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
		lmsUserID, now,
	).Order("banned_at DESC").First(&row).Error
	if err != nil {
		return nil
	}
	return &PublicBanInfo{
		Reason:      row.Reason,
		ExpiresAt:   row.ExpiresAt,
		IsPermanent: row.ExpiresAt == nil,
	}
}

// PublicBanJSON maps ban info for HTTP/JSON responses.
func PublicBanJSON(info *PublicBanInfo) map[string]interface{} {
	if info == nil {
		return nil
	}
	out := map[string]interface{}{
		"reason":      info.Reason,
		"isPermanent": info.IsPermanent,
		"expiresAt":   nil,
	}
	if info.ExpiresAt != nil {
		out["expiresAt"] = info.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}
