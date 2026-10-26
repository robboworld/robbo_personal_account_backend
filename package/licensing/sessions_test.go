package licensing

import (
	"sort"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// In-memory lk_user_sessions for a Free account (no licenses: session limit FreeSessionLimit).
type sessionGateway struct {
	Gateway
	rows  []*models.UserSessionCore
	clock time.Time
}

func (g *sessionGateway) ListLicensesByUser(string) ([]*models.LicenseCore, error) { return nil, nil }

func (g *sessionGateway) ListActiveSessions(string) ([]*models.UserSessionCore, error) {
	out := append([]*models.UserSessionCore(nil), g.rows...)
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeenAt.After(out[j].LastSeenAt) })
	return out, nil
}

func (g *sessionGateway) RevokeSession(key string) error {
	for i, r := range g.rows {
		if r.SessionKey == key {
			g.rows = append(g.rows[:i], g.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (g *sessionGateway) FindReusableSession(_, deviceID, ip string) (*models.UserSessionCore, error) {
	for _, r := range g.rows {
		if deviceID != "" && r.DeviceID == deviceID {
			return r, nil
		}
		if deviceID == "" && r.DeviceID == "" && ip != "" && r.IPAddress == ip {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *sessionGateway) ReuseSession(key, _, _, _, _ string, _, lastSeenAt time.Time) error {
	for _, r := range g.rows {
		if r.SessionKey == key {
			r.LastSeenAt = lastSeenAt
		}
	}
	return nil
}

func (g *sessionGateway) CreateSession(s *models.UserSessionCore) (*models.UserSessionCore, error) {
	s.SessionKey = s.DeviceID + "-key"
	// Distinct, increasing times: the real clock may not tick between sign-ins in a test.
	g.clock = g.clock.Add(time.Minute)
	s.LastSeenAt = g.clock
	g.rows = append(g.rows, s)
	return s, nil
}

func signIn(g Gateway, device, ip string) (*models.UserSessionCore, error) {
	return AcquireLoginSession(g, "42", "password",
		LoginClient{UserAgent: "test", IPAddress: ip, DeviceID: device}, time.Hour, models.Student)
}

func devices(g *sessionGateway) map[string]bool {
	out := map[string]bool{}
	for _, r := range g.rows {
		out[r.DeviceID] = true
	}
	return out
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
}

func TestSignInOverLimitEndsOldestSession(t *testing.T) {
	g := &sessionGateway{}
	for i := 0; i < FreeSessionLimit; i++ {
		if _, err := signIn(g, string(rune('a'+i)), "1.1.1.1"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := signIn(g, "new", "1.1.1.1"); err != nil {
		t.Fatalf("sign-in over the limit must succeed: %v", err)
	}
	d := devices(g)
	if len(g.rows) != FreeSessionLimit || d["a"] || !d["new"] {
		t.Fatalf("expected the oldest device signed out: rows=%d devices=%v", len(g.rows), d)
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

func TestClientWithoutCookieReusesSessionByIP(t *testing.T) {
	g := &sessionGateway{}
	first, _ := signIn(g, "", "1.1.1.1")
	again, err := signIn(g, "", "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if again.SessionKey != first.SessionKey || len(g.rows) != 1 {
		t.Fatalf("cookie-less repeat sign-in created a new session: rows=%d", len(g.rows))
	}
}
