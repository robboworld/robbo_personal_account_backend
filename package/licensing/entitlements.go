package licensing

import (
	"errors"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// Free/Standard tariff defaults, used when the account has no active license.
// SessionLimit=0 means unlimited (only paid tariffs enforce a concurrent-session cap).
const (
	FreeCloudQuotaMB = 10
	FreeSessionLimit = 0
)

// Entitlements is the resolved set of limits/capabilities for an account,
// derived from its active license (if any) or the Free/Standard defaults.
type Entitlements struct {
	HasLicense   bool
	TariffName   string
	CloudQuotaMB int
	SessionLimit int
	SeatLimit    int
	Capabilities []string
}

// ResolveEntitlements looks up the account's active, non-expired license and
// returns its limits; falls back to Free/Standard defaults when there is none.
func ResolveEntitlements(gateway Gateway, lmsUserID string) (Entitlements, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return freeEntitlements(), nil
	}
	licenses, err := gateway.ListLicensesByUser(lmsUserID)
	if err != nil {
		return Entitlements{}, err
	}
	now := time.Now().UTC()
	for _, lic := range licenses {
		if lic.Status == models.LicenseStatusActive && lic.ExpiresAt.After(now) {
			name := "Individual"
			if lic.SeatLimit >= 5 || lic.SessionLimit >= 5 || lic.CloudQuotaMB >= 500 {
				name = "Class"
			}
			return Entitlements{
				HasLicense:   true,
				TariffName:   name,
				CloudQuotaMB: lic.CloudQuotaMB,
				SessionLimit: lic.SessionLimit,
				SeatLimit:    lic.SeatLimit,
				Capabilities: lic.Capabilities,
			}, nil
		}
	}
	return freeEntitlements(), nil
}

func freeEntitlements() Entitlements {
	return Entitlements{
		HasLicense:   false,
		TariffName:   "Free",
		CloudQuotaMB: FreeCloudQuotaMB,
		SessionLimit: FreeSessionLimit,
	}
}

// CheckSessionLimit returns ErrSessionLimitReached when the account's active
// (non-revoked) session count has already reached its tariff's session_limit.
// SessionLimit<=0 means unlimited and is never enforced.
// Same client IP counts as a single session slot (see CountActiveSessions).
func CheckSessionLimit(gateway Gateway, lmsUserID string) error {
	entitlements, err := ResolveEntitlements(gateway, lmsUserID)
	if err != nil {
		return err
	}
	if entitlements.SessionLimit <= 0 {
		return nil
	}
	count, err := gateway.CountActiveSessions(lmsUserID)
	if err != nil {
		return err
	}
	if int(count) >= entitlements.SessionLimit {
		return ErrSessionLimitReached
	}
	return nil
}

// AcquireLoginSession reuses an active session for the same IP when present;
// otherwise enforces the tariff limit and creates a new row.
func AcquireLoginSession(
	gateway Gateway,
	lmsUserID, authMode, userAgent, ipAddress string,
	ttl time.Duration,
) (*models.UserSessionCore, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	ipAddress = strings.TrimSpace(ipAddress)
	if lmsUserID == "" {
		return nil, ErrBadRequest
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	if ipAddress != "" {
		existing, err := gateway.FindActiveSessionByIP(lmsUserID, ipAddress)
		if err == nil && existing != nil {
			if reuseErr := gateway.ReuseSession(existing.SessionKey, authMode, userAgent, expiresAt, now); reuseErr != nil {
				return nil, reuseErr
			}
			existing.AuthMode = authMode
			existing.UserAgent = userAgent
			existing.LastSeenAt = now
			existing.ExpiresAt = expiresAt
			existing.IPAddress = ipAddress
			return existing, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if err := CheckSessionLimit(gateway, lmsUserID); err != nil {
		return nil, err
	}
	return gateway.CreateSession(&models.UserSessionCore{
		LmsUserID:  lmsUserID,
		AuthMode:   authMode,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
	})
}
