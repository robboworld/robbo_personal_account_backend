//go:build integration

// Runs against a licensing database with feature_interest (migrations 0006/0007 or init/13):
//
//	LICENSING_TEST_DSN="host=... dbname=..." go test -tags integration ./package/licensing/gateway/
package gateway

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func interestTestGateway(t *testing.T) (*LicensingGatewayImpl, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("LICENSING_TEST_DSN")
	if dsn == "" {
		t.Skip("LICENSING_TEST_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return &LicensingGatewayImpl{db: db}, db
}

func TestListFeatureInterestNewestFirstAndPaged(t *testing.T) {
	g, db := interestTestGateway(t)
	prefix := "it-" + uuid.NewString()[:8]
	ids := []string{prefix + "-a", prefix + "-b", prefix + "-c"}
	t.Cleanup(func() { db.Exec(`DELETE FROM feature_interest WHERE lms_user_id LIKE ?`, prefix+"%") })

	for _, id := range ids {
		if err := g.SaveFeatureInterest(&models.FeatureInterestCore{
			LmsUserID: id, Role: "parent", Feature: "club_showcase", Org: id,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Dates far in the future put our rows on top whatever else the database holds:
	// a is the newest, c the oldest.
	base := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, id := range ids {
		if err := db.Exec(`UPDATE feature_interest SET updated_at = ? WHERE lms_user_id = ?`,
			base.Add(-time.Duration(i)*time.Hour), id).Error; err != nil {
			t.Fatal(err)
		}
	}

	rows, total, err := g.ListFeatureInterest(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total < 3 {
		t.Fatalf("total=%d, want at least our 3 rows", total)
	}
	if len(rows) != 2 || rows[0].Org != ids[0] || rows[1].Org != ids[1] {
		t.Fatalf("first page must be a, b newest first; got %+v", rows)
	}

	rows, _, err = g.ListFeatureInterest(2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 || rows[0].Org != ids[2] {
		t.Fatalf("second page must start with c; got %+v", rows)
	}
}

func TestListFeatureInterestRepeatAnswerReplacesNotDuplicates(t *testing.T) {
	g, db := interestTestGateway(t)
	id := "it-" + uuid.NewString()[:8]
	t.Cleanup(func() { db.Exec(`DELETE FROM feature_interest WHERE lms_user_id = ?`, id) })

	for _, pain := range []string{"first", "second"} {
		if err := g.SaveFeatureInterest(&models.FeatureInterestCore{
			LmsUserID: id, Role: "club_owner", Feature: "work_review", Pain: pain,
		}); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&models.FeatureInterestDB{}).Where("lms_user_id = ?", id).Count(&count)
	if count != 1 {
		t.Fatalf("one account must keep one row, got %d", count)
	}
	got, err := g.GetFeatureInterest(id)
	if err != nil || got.Pain != "second" {
		t.Fatalf("want the latest answer, got %+v err=%v", got, err)
	}
}
