package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
		DeviceID:   session.DeviceID,
		LastSeenAt: session.LastSeenAt,
		ExpiresAt:  session.ExpiresAt,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// CountActiveSessions counts active sessions; there is one per device (browser).
func (g *LicensingGatewayImpl) CountActiveSessions(lmsUserID string) (int64, error) {
	var count int64
	err := g.db.Model(&models.UserSessionDB{}).
		Where("lms_user_id = ? AND revoked_at IS NULL AND expires_at > ?", lmsUserID, time.Now().UTC()).
		Count(&count).Error
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

// FindReusableSession returns the active session of this device (browser). Without one,
// it falls back to a legacy row from before device tracking (empty device_id) with the same
// IP, so sessions opened before the upgrade are adopted rather than counted twice.
func (g *LicensingGatewayImpl) FindReusableSession(lmsUserID, deviceID, ipAddress string) (*models.UserSessionCore, error) {
	if lmsUserID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	var row models.UserSessionDB
	if deviceID != "" {
		err := g.db.Where(
			"lms_user_id = ? AND device_id = ? AND revoked_at IS NULL AND expires_at > ?",
			lmsUserID, deviceID, now,
		).Order("last_seen_at DESC").First(&row).Error
		if err == nil {
			return row.ToCore(), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return nil, gorm.ErrRecordNotFound
	}
	err := g.db.Where(
		"lms_user_id = ? AND device_id = '' AND ip_address = ? AND revoked_at IS NULL AND expires_at > ?",
		lmsUserID, ipAddress, now,
	).Order("last_seen_at DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

// ReuseSession refreshes an existing active session on a repeat sign-in from the same
// device; deviceID (when set) also claims a legacy row for that device.
func (g *LicensingGatewayImpl) ReuseSession(sessionKey, authMode, userAgent, ipAddress, deviceID string, expiresAt, lastSeenAt time.Time) error {
	if sessionKey == "" {
		return nil
	}
	updates := map[string]interface{}{
		"auth_mode":    authMode,
		"user_agent":   userAgent,
		"last_seen_at": lastSeenAt,
		"expires_at":   expiresAt,
	}
	if ipAddress != "" {
		updates["ip_address"] = ipAddress
	}
	if deviceID != "" {
		updates["device_id"] = deviceID
	}
	return g.db.Model(&models.UserSessionDB{}).
		Where("session_key = ? AND revoked_at IS NULL AND expires_at > ?", sessionKey, time.Now().UTC()).
		Updates(updates).Error
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

// RevokeAllSessionsForUser marks every non-revoked session for the user as revoked.
// Returns the number of rows updated.
func (g *LicensingGatewayImpl) RevokeAllSessionsForUser(lmsUserID string) (int, error) {
	if lmsUserID == "" {
		return 0, nil
	}
	now := time.Now().UTC()
	res := g.db.Model(&models.UserSessionDB{}).
		Where("lms_user_id = ? AND revoked_at IS NULL", lmsUserID).
		Update("revoked_at", now)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
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
