package licensing

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// In-memory lk_user_sessions for a Free account (no licenses: session limit 2).
type sessionGateway struct {
	Gateway
	rows []*models.UserSessionCore
}

func (g *sessionGateway) ListLicensesByUser(string) ([]*models.LicenseCore, error) { return nil, nil }

func (g *sessionGateway) CountActiveSessions(string) (int64, error) {
	return int64(len(g.rows)), nil
}

func (g *sessionGateway) FindReusableSession(_, deviceID, ip string) (*models.UserSessionCore, error) {
	for _, r := range g.rows {
		if deviceID != "" && r.DeviceID == deviceID {
			return r, nil
		}
	}
	for _, r := range g.rows {
		if r.DeviceID == "" && ip != "" && r.IPAddress == ip {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *sessionGateway) ReuseSession(key, _, _, ip, deviceID string, _, _ time.Time) error {
	for _, r := range g.rows {
		if r.SessionKey == key && deviceID != "" {
			r.DeviceID = deviceID
		}
	}
	return nil
}

func (g *sessionGateway) CreateSession(s *models.UserSessionCore) (*models.UserSessionCore, error) {
	s.SessionKey = s.DeviceID + "-key"
	g.rows = append(g.rows, s)
	return s, nil
}

func signIn(g Gateway, device, ip string) (*models.UserSessionCore, error) {
	return AcquireLoginSession(g, "42", "password",
		LoginClient{UserAgent: "test", IPAddress: ip, DeviceID: device}, time.Hour, models.Student)
}

func TestSessionsAreCountedPerDeviceNotPerIP(t *testing.T) {
	g := &sessionGateway{}
	const schoolNAT = "10.0.0.1"

	if _, err := signIn(g, "pc1", schoolNAT); err != nil {
		t.Fatal(err)
	}
	if _, err := signIn(g, "pc2", schoolNAT); err != nil {
		t.Fatal(err)
	}
	if len(g.rows) != 2 {
		t.Fatalf("two PCs behind one IP must take two slots, got %d", len(g.rows))
	}
	if _, err := signIn(g, "pc3", schoolNAT); !errors.Is(err, ErrSessionLimitReached) {
		t.Fatalf("third PC on Free (limit 2): err=%v", err)
	}
}

func TestRepeatSignInFromSameDeviceReusesSession(t *testing.T) {
	g := &sessionGateway{}
	first, _ := signIn(g, "laptop", "1.1.1.1")
	again, err := signIn(g, "laptop", "2.2.2.2") // same browser, another network
	if err != nil {
		t.Fatal(err)
	}
	if again.SessionKey != first.SessionKey || len(g.rows) != 1 {
		t.Fatalf("repeat sign-in created a new session: rows=%d", len(g.rows))
	}
}

func TestLegacySessionIsAdoptedByDevice(t *testing.T) {
	g := &sessionGateway{rows: []*models.UserSessionCore{
		{SessionKey: "legacy", IPAddress: "1.1.1.1"}, // from before device tracking
	}}
	sess, err := signIn(g, "laptop", "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if sess.SessionKey != "legacy" || len(g.rows) != 1 || g.rows[0].DeviceID != "laptop" {
		t.Fatalf("legacy row not adopted: key=%s rows=%d", sess.SessionKey, len(g.rows))
	}
}
