package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

const mb = int64(1) << 20

type quotaPageGateway struct {
	projectPage.Gateway
	used, current int64
}

func (g *quotaPageGateway) GetTotalStorageBytesForOwner(string) (int64, error) { return g.used, nil }
func (g *quotaPageGateway) GetCurrentVersionSizeBytes(string) (int64, error)   { return g.current, nil }

func quotaUseCase(used, current int64, licenses ...*models.LicenseCore) *ProjectPageUseCaseImpl {
	return &ProjectPageUseCaseImpl{
		projectPageGateway: &quotaPageGateway{used: used, current: current},
		licensingGateway:   &quotaLicensingGateway{licenses: licenses},
	}
}

func TestStorageQuotaFreeAccount(t *testing.T) {
	// Free: 100 MB in total.
	if err := quotaUseCase(99*mb, 0).enforceStorageQuota("1", "", 0); err != nil {
		t.Fatalf("new project under the quota: %v", err)
	}
	if err := quotaUseCase(100*mb, 0).enforceStorageQuota("1", "", 0); !errors.Is(err, projectPage.ErrStorageQuotaExceeded) {
		t.Fatalf("new project with the quota used up: err=%v", err)
	}
	if err := quotaUseCase(99*mb, 1*mb).enforceStorageQuota("1", "p", 3*mb); !errors.Is(err, projectPage.ErrStorageQuotaExceeded) {
		t.Fatalf("growth past the quota: err=%v", err)
	}
	if err := quotaUseCase(98*mb, 1*mb).enforceStorageQuota("1", "p", 3*mb); err != nil {
		t.Fatalf("growth up to exactly the quota: %v", err)
	}
}

func TestStorageQuotaNeverBlocksShrinkingOverQuota(t *testing.T) {
	// 1.5 GB left from an expired license, now Free (100 MB).
	uc := quotaUseCase(1536*mb, 10*mb)
	if err := uc.enforceStorageQuota("1", "p", 5*mb); err != nil {
		t.Fatalf("shrinking save refused: %v", err)
	}
	if err := uc.enforceStorageQuota("1", "p", 10*mb); err != nil {
		t.Fatalf("same-size save refused: %v", err)
	}
	if err := uc.enforceStorageQuota("1", "p", 11*mb); !errors.Is(err, projectPage.ErrStorageQuotaExceeded) {
		t.Fatalf("growth over the quota: err=%v", err)
	}
}

func TestStorageQuotaFromLicense(t *testing.T) {
	class := &models.LicenseCore{
		Status:         models.LicenseStatusActive,
		ExpiresAt:      time.Now().Add(time.Hour),
		CloudQuotaMB:   50,
		StorageQuotaMB: 5120,
	}
	if err := quotaUseCase(4000*mb, 0, class).enforceStorageQuota("1", "p", 50*mb); err != nil {
		t.Fatalf("Class account under 5 GB: %v", err)
	}
	if err := quotaUseCase(5100*mb, 0, class).enforceStorageQuota("1", "p", 50*mb); !errors.Is(err, projectPage.ErrStorageQuotaExceeded) {
		t.Fatalf("Class account over 5 GB: err=%v", err)
	}
}
