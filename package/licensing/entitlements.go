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
// SessionLimit=0 / SeatLimit=0 means unlimited. The number of projects is not limited:
// the account's StorageQuotaMB bounds it.
// FreeCloudQuotaMB is the max size of a single .sb3 project; FreeStorageQuotaMB is the
// total for the whole account (all projects; later also cloud files).
// Free has no seat limit: seats are RS3 Desktop activations by license key, and a Free
// account has no license to activate (Entitlements.SeatLimit stays 0, HasLicense false).
const (
	FreeCloudQuotaMB   = 10
	FreeStorageQuotaMB = 100
	FreeSessionLimit   = 5
)

// DefaultLicenseStorageQuotaMB is the account total for a license or product created
// without one (admin grants); it matches the Individual tariff.
const DefaultLicenseStorageQuotaMB = 1024

// Entitlements is the resolved set of limits/capabilities for an account,
// derived from its active license (if any) or the Free/Standard defaults.
type Entitlements struct {
	HasLicense       bool
	TariffName       string
	CloudQuotaMB     int // alias / legacy: same as MaxProjectSizeMB
	MaxProjectSizeMB int // max .sb3 size per project in MB
	StorageQuotaMB   int // total for all the account's projects in MB
	SessionLimit     int
	SeatLimit        int
	Capabilities     []string
	// ExpiresAt of the active licence; zero for Free.
	ExpiresAt time.Time
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
			storageMB := lic.StorageQuotaMB
			if storageMB <= 0 {
				storageMB = DefaultLicenseStorageQuotaMB
			}
			return Entitlements{
				HasLicense:       true,
				TariffName:       name,
				CloudQuotaMB:     sizeMB,
				MaxProjectSizeMB: sizeMB,
				StorageQuotaMB:   storageMB,
				SessionLimit:     lic.SessionLimit,
				SeatLimit:        lic.SeatLimit,
				Capabilities:     lic.Capabilities,
				ExpiresAt:        lic.ExpiresAt,
			}, nil
		}
	}
	return freeEntitlements(), nil
}

// TariffName is the licence's tariff as the ЛК shows it: the product title, or Individual / Class
// by its limits.
func TariffName(gateway Gateway, lic *models.LicenseCore) string {
	return resolveTariffName(gateway, lic)
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
	if IsClassLicense(lic) {
		return "Class"
	}
	return "Individual"
}

// IsClassLicense: a licence for a class or a school (many computers or sessions), not one person.
func IsClassLicense(lic *models.LicenseCore) bool {
	return lic != nil && (lic.SeatLimit >= 5 || lic.SessionLimit >= 5 || lic.CloudQuotaMB >= 500)
}

func freeEntitlements() Entitlements {
	return Entitlements{
		HasLicense:       false,
		TariffName:       "Free",
		CloudQuotaMB:     FreeCloudQuotaMB,
		MaxProjectSizeMB: FreeCloudQuotaMB,
		StorageQuotaMB:   FreeStorageQuotaMB,
		SessionLimit:     FreeSessionLimit,
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

// LMSUserDisplayName is the name of an LMS account as people see it: the profile's full name,
// the first and last name, or the username. Empty when the LMS cannot be read.
func LMSUserDisplayName(lmsUserID string) string {
	id, err := strconv.ParseInt(strings.TrimSpace(lmsUserID), 10, 64)
	if err != nil || id <= 0 {
		return ""
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		log.Printf("licensing: LMS reader for the licence owner name: %v", err)
		return ""
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return ""
	}
	if name := strings.TrimSpace(profile.ProfileName); name != "" {
		return name
	}
	if name := strings.TrimSpace(profile.FirstName + " " + profile.LastName); name != "" {
		return name
	}
	return strings.TrimSpace(profile.Username)
}

// makeRoomForSession ends the account's least recently used sessions so one more fits its
// session limit: signing in on another device signs out the oldest one instead of failing.
func makeRoomForSession(gateway Gateway, lmsUserID string, role models.Role) error {
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
	sessions, err := gateway.ListActiveSessions(lmsUserID) // most recently used first
	if err != nil {
		return err
	}
	for i := entitlements.SessionLimit - 1; i < len(sessions); i++ {
		if err := gateway.RevokeSession(sessions[i].SessionKey); err != nil {
			return err
		}
	}
	return nil
}

// KickOtherSessions revokes every non-revoked web session for the user ("sign out on all
// devices" at sign-in).
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

// LoginClient describes where a sign-in comes from. DeviceID is the lk_device cookie
// (one per browser); IPAddress is kept for "My sessions" and legacy-row adoption.
type LoginClient struct {
	UserAgent string
	IPAddress string
	DeviceID  string
}

// AcquireLoginSession reuses the device's active session when present; otherwise
// enforces the tariff limit and creates a new row for the device.
// role is used to exempt SuperAdmin / UnitAdmin from the concurrent-session cap.
func AcquireLoginSession(
	gateway Gateway,
	lmsUserID, authMode string,
	client LoginClient,
	ttl time.Duration,
	role models.Role,
) (*models.UserSessionCore, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	ipAddress := strings.TrimSpace(client.IPAddress)
	deviceID := strings.TrimSpace(client.DeviceID)
	if lmsUserID == "" {
		return nil, ErrBadRequest
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	existing, err := gateway.FindReusableSession(lmsUserID, deviceID, ipAddress)
	if err == nil && existing != nil {
		if reuseErr := gateway.ReuseSession(
			existing.SessionKey, authMode, client.UserAgent, ipAddress, deviceID, expiresAt, now,
		); reuseErr != nil {
			return nil, reuseErr
		}
		existing.AuthMode = authMode
		existing.UserAgent = client.UserAgent
		existing.LastSeenAt = now
		existing.ExpiresAt = expiresAt
		if ipAddress != "" {
			existing.IPAddress = ipAddress
		}
		if deviceID != "" {
			existing.DeviceID = deviceID
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := makeRoomForSession(gateway, lmsUserID, role); err != nil {
		return nil, err
	}
	return gateway.CreateSession(&models.UserSessionCore{
		LmsUserID:  lmsUserID,
		AuthMode:   authMode,
		UserAgent:  client.UserAgent,
		IPAddress:  ipAddress,
		DeviceID:   deviceID,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
	})
}
