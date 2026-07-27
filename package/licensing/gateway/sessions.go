package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession inserts a new active web login session row. Generates a
// random session_key when the caller did not already set one.
func (g *LicensingGatewayImpl) CreateSession(session *models.UserSessionCore) (*models.UserSessionCore, error) {
	if session.SessionKey == "" {
		key, err := randomHex(24)
		if err != nil {
			return nil, err
		}
		session.SessionKey = key
	}
	now := time.Now().UTC()
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}
	row := models.UserSessionDB{
		LmsUserID:  session.LmsUserID,
		SessionKey: session.SessionKey,
		AuthMode:   session.AuthMode,
		UserAgent:  session.UserAgent,
		IPAddress:  session.IPAddress,
		LastSeenAt: session.LastSeenAt,
		ExpiresAt:  session.ExpiresAt,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// CountActiveSessions counts distinct client IPs among active sessions.
// Empty IP is counted per-row (each blank IP is its own slot).
func (g *LicensingGatewayImpl) CountActiveSessions(lmsUserID string) (int64, error) {
	var count int64
	err := g.db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT DISTINCT CASE
				WHEN COALESCE(TRIM(ip_address), '') = '' THEN id::text
				ELSE TRIM(ip_address)
			END AS slot
			FROM lk_user_sessions
			WHERE lms_user_id = ? AND revoked_at IS NULL AND expires_at > ?
		) slots`,
		lmsUserID, time.Now().UTC(),
	).Scan(&count).Error
	return count, err
}

// GetActiveSession returns a non-revoked, non-expired session by session_key.
func (g *LicensingGatewayImpl) GetActiveSession(sessionKey string) (*models.UserSessionCore, error) {
	if sessionKey == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.UserSessionDB
	err := g.db.Where(
		"session_key = ? AND revoked_at IS NULL AND expires_at > ?",
		sessionKey, time.Now().UTC(),
	).First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// FindActiveSessionByIP returns the most recently seen active session for this IP.
func (g *LicensingGatewayImpl) FindActiveSessionByIP(lmsUserID, ipAddress string) (*models.UserSessionCore, error) {
	ipAddress = strings.TrimSpace(ipAddress)
	if lmsUserID == "" || ipAddress == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.UserSessionDB
	err := g.db.Where(
		"lms_user_id = ? AND ip_address = ? AND revoked_at IS NULL AND expires_at > ?",
		lmsUserID, ipAddress, time.Now().UTC(),
	).Order("last_seen_at DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// ReuseSession refreshes metadata on an existing active session (same IP login).
func (g *LicensingGatewayImpl) ReuseSession(sessionKey, authMode, userAgent string, expiresAt, lastSeenAt time.Time) error {
	if sessionKey == "" {
		return nil
	}
	return g.db.Model(&models.UserSessionDB{}).
		Where("session_key = ? AND revoked_at IS NULL AND expires_at > ?", sessionKey, time.Now().UTC()).
		Updates(map[string]interface{}{
			"auth_mode":    authMode,
			"user_agent":   userAgent,
			"last_seen_at": lastSeenAt,
			"expires_at":   expiresAt,
		}).Error
}

// TouchSession bumps last_seen_at, e.g. on access-token refresh. No-op (not
// an error) when the session no longer exists.
func (g *LicensingGatewayImpl) TouchSession(sessionKey string, lastSeenAt time.Time) error {
	if sessionKey == "" {
		return nil
	}
	return g.db.Model(&models.UserSessionDB{}).
		Where("session_key = ? AND revoked_at IS NULL AND expires_at > ?", sessionKey, time.Now().UTC()).
		Update("last_seen_at", lastSeenAt).Error
}

// RevokeSession marks a session revoked (sign-out / OIDC logout). No-op when
// the session key is empty or already gone.
func (g *LicensingGatewayImpl) RevokeSession(sessionKey string) error {
	if sessionKey == "" {
		return nil
	}
	now := time.Now().UTC()
	return g.db.Model(&models.UserSessionDB{}).
		Where("session_key = ? AND revoked_at IS NULL", sessionKey).
		Update("revoked_at", now).Error
}

// RevokeSessionByID revokes a session by UUID, ownership-checked against lmsUserID.
func (g *LicensingGatewayImpl) RevokeSessionByID(lmsUserID, sessionID string) error {
	if lmsUserID == "" || sessionID == "" {
		return nil
	}
	now := time.Now().UTC()
	res := g.db.Model(&models.UserSessionDB{}).
		Where("id = ? AND lms_user_id = ? AND revoked_at IS NULL", sessionID, lmsUserID).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListActiveSessions returns non-revoked, non-expired sessions for the "My
// sessions" UI, most recently seen first.
func (g *LicensingGatewayImpl) ListActiveSessions(lmsUserID string) ([]*models.UserSessionCore, error) {
	var rows []models.UserSessionDB
	err := g.db.Where("lms_user_id = ? AND revoked_at IS NULL AND expires_at > ?", lmsUserID, time.Now().UTC()).
		Order("last_seen_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*models.UserSessionCore, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToCore())
	}
	return out, nil
}
