package storage

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

const mb = int64(1) << 20

type fixedUsage int64

func (u fixedUsage) GetTotalStorageBytesForOwner(string) (int64, error) { return int64(u), nil }

type fakeLicensing struct {
	licensing.Gateway
	licenses []*models.LicenseCore
}

func (g *fakeLicensing) ListLicensesByUser(string) ([]*models.LicenseCore, error) {
	return g.licenses, nil
}
func (g *fakeLicensing) GetProductTitle(string) (string, error) { return "", nil }

func TestQuotaSumsProjectsAndFiles(t *testing.T) {
	// Free: 100 MB for projects and files together.
	q := NewQuota(&fakeLicensing{}, fixedUsage(60*mb), fixedUsage(30*mb))
	u, err := q.Usage("1")
	if err != nil || u.Total() != 90*mb {
		t.Fatalf("usage=%+v err=%v, want 90 MB", u, err)
	}
	if err := q.AllowGrowth("1", 0, 10*mb); err != nil {
		t.Fatalf("a file up to exactly the quota: %v", err)
	}
	if err := q.AllowGrowth("1", 0, 11*mb); !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("a file past the quota counted with projects: err=%v", err)
	}
	if err := q.AllowNewItem("1"); err != nil {
		t.Fatalf("new project with 10 MB left: %v", err)
	}
	if err := NewQuota(&fakeLicensing{}, fixedUsage(70*mb), fixedUsage(30*mb)).AllowNewItem("1"); !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("new project with the quota used up by files: err=%v", err)
	}
}

func TestQuotaNeverBlocksShrinking(t *testing.T) {
	q := NewQuota(&fakeLicensing{}, fixedUsage(1536*mb), nil)
	if err := q.AllowGrowth("1", 10*mb, 10*mb); err != nil {
		t.Fatalf("same-size save over the quota refused: %v", err)
	}
	if err := q.AllowGrowth("1", 10*mb, 11*mb); !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("growth over the quota: err=%v", err)
	}
}

func TestQuotaFromLicence(t *testing.T) {
	class := &models.LicenseCore{
		Status:         models.LicenseStatusActive,
		ExpiresAt:      time.Now().Add(time.Hour),
		CloudQuotaMB:   50,
		StorageQuotaMB: 5120,
	}
	q := NewQuota(&fakeLicensing{licenses: []*models.LicenseCore{class}}, fixedUsage(4000*mb), fixedUsage(1000*mb))
	if err := q.AllowGrowth("1", 0, 50*mb); err != nil {
		t.Fatalf("Class under 5 GB: %v", err)
	}
	if err := q.AllowGrowth("1", 0, 121*mb); !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("Class over 5 GB: err=%v", err)
	}
}

func TestQuotaWithoutLicensingIsUnlimited(t *testing.T) {
	var nilQuota *Quota
	if err := nilQuota.AllowGrowth("1", 0, 1<<40); err != nil {
		t.Fatalf("nil quota: %v", err)
	}
	if err := NewQuota(nil, fixedUsage(1<<40), nil).AllowNewItem("1"); err != nil {
		t.Fatalf("no licensing gateway: %v", err)
	}
}
