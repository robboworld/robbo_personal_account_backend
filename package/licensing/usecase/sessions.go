package usecase

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

const (
	AuthModeLegacyJWT = "legacy_jwt"
	AuthModeLmsDB     = "lms_db"
	AuthModeOidcBFF   = "oidc_bff"
)

// BeginLoginSession checks the concurrent-session tariff limit, then inserts a
// new lk_user_sessions row. Returns the session_key to embed as JWT/BFF sid.
func (u *LicensingUseCaseImpl) BeginLoginSession(
	lmsUserID, authMode, userAgent, ipAddress string,
	ttl time.Duration,
) (*models.UserSessionCore, error) {
	if lmsUserID == "" {
		return nil, licensing.ErrBadRequest
	}
	if err := licensing.CheckSessionLimit(u.gateway, lmsUserID); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	return u.gateway.CreateSession(&models.UserSessionCore{
		LmsUserID:  lmsUserID,
		AuthMode:   authMode,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		LastSeenAt: now,
		ExpiresAt:  now.Add(ttl),
	})
}

// TouchLoginSession bumps last_seen_at for an active session. Returns
// ErrSessionNotFound when the row is missing, revoked, or expired.
func (u *LicensingUseCaseImpl) TouchLoginSession(sessionKey string) error {
	if sessionKey == "" {
		return nil
	}
	sess, err := u.gateway.GetActiveSession(sessionKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return licensing.ErrSessionNotFound
		}
		return err
	}
	if sess == nil {
		return licensing.ErrSessionNotFound
	}
	return u.gateway.TouchSession(sessionKey, time.Now().UTC())
}

// RevokeLoginSession best-effort revokes by session_key (sign-out / logout).
func (u *LicensingUseCaseImpl) RevokeLoginSession(sessionKey string) error {
	return u.gateway.RevokeSession(sessionKey)
}

// ListLoginSessions returns active sessions for the "My sessions" UI.
func (u *LicensingUseCaseImpl) ListLoginSessions(lmsUserID string) ([]*models.UserSessionCore, error) {
	return u.gateway.ListActiveSessions(lmsUserID)
}

// RevokeLoginSessionByID revokes one session owned by lmsUserID.
func (u *LicensingUseCaseImpl) RevokeLoginSessionByID(lmsUserID, sessionID string) error {
	err := u.gateway.RevokeSessionByID(lmsUserID, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return licensing.ErrSessionNotFound
	}
	return err
}

// ResolveEntitlements returns effective tariff limits for the account.
func (u *LicensingUseCaseImpl) ResolveEntitlements(lmsUserID string) (licensing.Entitlements, error) {
	return licensing.ResolveEntitlements(u.gateway, lmsUserID)
}
