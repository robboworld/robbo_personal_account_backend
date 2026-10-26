package usecase

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/spf13/viper"
)

// Version counts per project; pruning trims a project down to policy.KeepLast.
type retentionGateway struct {
	projectPage.Gateway
	versions map[string]int
	noop     bool
}

func (g *retentionGateway) ListSb3PruneCandidates(policy projectPage.Sb3RetentionPolicy, limit int) ([]string, error) {
	var ids []string
	for id, n := range g.versions {
		if n > policy.KeepLast {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (g *retentionGateway) PruneSb3Versions(projectID string, policy projectPage.Sb3RetentionPolicy) (int64, error) {
	if g.noop {
		return 0, nil
	}
	n := g.versions[projectID] - policy.KeepLast
	g.versions[projectID] = policy.KeepLast
	return int64(n), nil
}

func TestPruneSb3VersionsDrainsAllBatches(t *testing.T) {
	versions := map[string]int{"single": 1}
	for i := 0; i < pruneCandidatesBatch+10; i++ {
		versions[fmt.Sprintf("p%03d", i)] = 100
	}
	gw := &retentionGateway{versions: versions}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}

	deleted, err := uc.PruneSb3Versions(projectPage.Sb3RetentionPolicy{KeepLast: 1})
	if err != nil {
		t.Fatal(err)
	}
	if want := int64((pruneCandidatesBatch + 10) * 99); deleted != want {
		t.Fatalf("deleted=%d, want %d", deleted, want)
	}
	if versions["single"] != 1 {
		t.Fatalf("project with one version was touched: %d left", versions["single"])
	}
}

func TestPruneSb3VersionsStopsWhenNothingDeleted(t *testing.T) {
	gw := &retentionGateway{versions: map[string]int{"p": 100}, noop: true}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: gw}

	deleted, err := uc.PruneSb3Versions(projectPage.Sb3RetentionPolicy{KeepLast: 1})
	if err != nil || deleted != 0 {
		t.Fatalf("deleted=%d err=%v", deleted, err)
	}
}

func TestRetentionDefaultsKeepOnlyLatest(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	if !retentionEnabled() {
		t.Fatal("retention must be on by default")
	}
	if got := retentionPolicyFromConfig(); got.KeepLast != 1 || got.KeepDailyDays != 0 {
		t.Fatalf("defaults = %+v, want only the latest version", got)
	}

	viper.Set("projectPage.versionRetention.enabled", false)
	if retentionEnabled() {
		t.Fatal("explicit enabled: false must turn retention off")
	}

	viper.Set("projectPage.versionRetention.keepLast", 5)
	viper.Set("projectPage.versionRetention.keepDailyDays", 7)
	if got := retentionPolicyFromConfig(); got.KeepLast != 5 || got.KeepDailyDays != 7 {
		t.Fatalf("configured history = %+v", got)
	}
}

type quotaLicensingGateway struct {
	licensing.Gateway
	licenses []*models.LicenseCore
}

func (g *quotaLicensingGateway) ListLicensesByUser(string) ([]*models.LicenseCore, error) {
	return g.licenses, nil
}

func (g *quotaLicensingGateway) GetProductTitle(string) (string, error) {
	return "", nil
}

func TestMaxSb3UploadBytes(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	uc := &ProjectPageUseCaseImpl{}
	if got, _ := uc.MaxSb3UploadBytes("1"); got != defaultMaxUploadMB<<20 {
		t.Fatalf("no licensing: got %d, want ceiling", got)
	}

	uc.licensingGateway = &quotaLicensingGateway{}
	if got, _ := uc.MaxSb3UploadBytes("1"); got != licensing.FreeCloudQuotaMB<<20 {
		t.Fatalf("free tariff: got %d", got)
	}

	uc.licensingGateway = &quotaLicensingGateway{licenses: []*models.LicenseCore{{
		Status:       models.LicenseStatusActive,
		ExpiresAt:    time.Now().Add(time.Hour),
		CloudQuotaMB: 500,
	}}}
	viper.Set("projectPage.maxUploadMB", 100)
	if got, _ := uc.MaxSb3UploadBytes("1"); got != 100<<20 {
		t.Fatalf("tariff above ceiling: got %d, want 100 MiB", got)
	}
}
