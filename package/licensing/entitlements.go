package licensing

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// Free/Standard tariff defaults, used when the account has no active license.
// SessionLimit=0 / SeatLimit=0 / MaxProjects=0 means unlimited.
// FreeCloudQuotaMB is the max size of a single .sb3 project (not a total storage quota).
const (
	FreeCloudQuotaMB = 10
	FreeSessionLimit = 1
	FreeSeatLimit    = 1
	FreeMaxProjects  = 20
)

// Entitlements is the resolved set of limits/capabilities for an account,
// derived from its active license (if any) or the Free/Standard defaults.
type Entitlements struct {
	HasLicense       bool
	TariffName       string
	CloudQuotaMB     int // alias / legacy: same as MaxProjectSizeMB
	MaxProjectSizeMB int // max .sb3 size per project in MB
	MaxProjects      int // max total projects (published + drafts); 0 = unlimited
	SessionLimit     int
	SeatLimit        int
	Capabilities     []string
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
			name := resolveTariffName(gateway, lic)
			sizeMB := lic.CloudQuotaMB
			if sizeMB <= 0 {
				sizeMB = FreeCloudQuotaMB
			}
			return Entitlements{
				HasLicense:       true,
				TariffName:       name,
				CloudQuotaMB:     sizeMB,
				MaxProjectSizeMB: sizeMB,
				MaxProjects:      0, // unlimited for paid
				SessionLimit:     lic.SessionLimit,
				SeatLimit:        lic.SeatLimit,
				Capabilities:     lic.Capabilities,
			}, nil
		}
	}
	return freeEntitlements(), nil
}

func resolveTariffName(gateway Gateway, lic *models.LicenseCore) string {
	if lic == nil {
		return "Individual"
	}
	if lic.ProductID != "" {
		if title, err := gateway.GetProductTitle(lic.ProductID); err == nil && strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
	}
	if lic.SeatLimit >= 5 || lic.SessionLimit >= 5 || lic.CloudQuotaMB >= 500 {
		return "Class"
	}
	return "Individual"
}

func freeEntitlements() Entitlements {
	return Entitlements{
		HasLicense:       false,
		TariffName:       "Free",
		CloudQuotaMB:     FreeCloudQuotaMB,
		MaxProjectSizeMB: FreeCloudQuotaMB,
		MaxProjects:      FreeMaxProjects,
		SessionLimit:     FreeSessionLimit,
		SeatLimit:        FreeSeatLimit,
	}
}

// UnlimitedSessionsAndSeats is true for SuperAdmin / UnitAdmin — no concurrent
// web-session or device-seat tariff caps.
func UnlimitedSessionsAndSeats(role models.Role) bool {
	return role == models.SuperAdmin || role == models.UnitAdmin
}

// ApplyAdminSessionSeatExemption clears session/seat caps for admin roles
// (0 = unlimited in entitlements / UI meters).
func ApplyAdminSessionSeatExemption(ent *Entitlements, role models.Role) {
	if ent == nil || !UnlimitedSessionsAndSeats(role) {
		return
	}
	ent.SessionLimit = 0
	ent.SeatLimit = 0
}

// LMSUserHasUnlimitedSessionsAndSeats looks up LMS auth_user flags for the
// given edx user id. Superuser → SuperAdmin exemption; failures fail closed
// (limits still apply).
func LMSUserHasUnlimitedSessionsAndSeats(lmsUserID string) bool {
	id, err := strconv.ParseInt(strings.TrimSpace(lmsUserID), 10, 64)
	if err != nil || id <= 0 {
		return false
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		log.Printf("licensing: LMS reader for admin seat/session exemption: %v", err)
		return false
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return false
	}
	if profile.IsSuperuser {
		return true
	}
	return false
}

// CheckSessionLimit returns ErrSessionLimitReached when the account's active
// (non-revoked) session count has already reached its tariff's session_limit.
// SessionLimit<=0 means unlimited and is never enforced.
// Admins (SuperAdmin / UnitAdmin) are always exempt.
// Same client IP counts as a single session slot (see CountActiveSessions).
func CheckSessionLimit(gateway Gateway, lmsUserID string, role models.Role) error {
	if UnlimitedSessionsAndSeats(role) {
		return nil
	}
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

// KickOtherSessions revokes every non-revoked web session for the user so a
// subsequent AcquireLoginSession can succeed after session_limit_reached.
func KickOtherSessions(gateway Gateway, lmsUserID string) error {
	if gateway == nil {
		return nil
	}
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return nil
	}
	_, err := gateway.RevokeAllSessionsForUser(lmsUserID)
	return err
}

// AcquireLoginSession reuses an active session for the same IP when present;
// otherwise enforces the tariff limit and creates a new row.
// role is used to exempt SuperAdmin / UnitAdmin from the concurrent-session cap.
func AcquireLoginSession(
	gateway Gateway,
	lmsUserID, authMode, userAgent, ipAddress string,
	ttl time.Duration,
	role models.Role,
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

	if err := CheckSessionLimit(gateway, lmsUserID, role); err != nil {
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
