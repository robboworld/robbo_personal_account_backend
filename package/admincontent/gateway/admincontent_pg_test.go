//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	ADMINCONTENT_TEST_DSN="host=... dbname=..." go test -tags integration ./package/admincontent/gateway/
package gateway

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ADMINCONTENT_TEST_DSN")
	if dsn == "" {
		t.Skip("ADMINCONTENT_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

// project inserts a project whose current version has size bytes (no version when size < 0).
func project(t *testing.T, db *gorm.DB, owner string, size int64, deleted bool) string {
	t.Helper()
	id := uuid.NewString()
	if err := db.Exec(`INSERT INTO scratch_projects (id, owner_user_id, title, deleted_at)
		VALUES (?, ?, 'p', CASE WHEN ? THEN now() END)`, id, owner, deleted).Error; err != nil {
		t.Fatal(err)
	}
	if size >= 0 {
		vid := uuid.NewString()
		sum := sha256.Sum256([]byte(vid))
		if err := db.Exec(`INSERT INTO scratch_project_versions
			(id, project_id, version_seq, archive, size_bytes, checksum_sha256, created_by_user_id)
			VALUES (?, ?, 1, '\x00', ?, ?, ?)`, vid, id, size, sum[:], owner).Error; err != nil {
			t.Fatal(err)
		}
		db.Exec(`UPDATE scratch_projects SET current_version_id = ? WHERE id = ?`, vid, id)
	}
	return id
}

func file(t *testing.T, db *gorm.DB, owner string, size int64, deleted bool) {
	t.Helper()
	if err := db.Exec(`INSERT INTO user_files (id, owner_user_id, original_name, kind, mime, size_bytes, sha256, deleted_at)
		VALUES (?, ?, 'a.txt', 'txt', 'text/plain', ?, repeat('0', 64), CASE WHEN ? THEN now() END)`,
		uuid.NewString(), owner, size, deleted).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAggregates(t *testing.T) {
	db := testDB(t)
	g := NewAdminContentGateway(db, db)
	a, b, c := "it-"+uuid.NewString(), "it-"+uuid.NewString(), "it-"+uuid.NewString()
	before, err := g.StorageTotals()
	if err != nil {
		t.Fatal(err)
	}
	project(t, db, a, 100, false)
	project(t, db, a, 50, false)
	project(t, db, a, 999, true) // deleted: not counted
	project(t, db, a, -1, false) // never saved: counted, 0 bytes
	file(t, db, a, 10, false)
	file(t, db, b, 500, false) // files only
	file(t, db, b, 777, true)  // deleted: not counted
	project(t, db, c, 30, false)

	after, err := g.StorageTotals()
	if err != nil {
		t.Fatal(err)
	}
	if d := after.ProjectsBytes - before.ProjectsBytes; d != 180 {
		t.Errorf("projects bytes +%d, want 180", d)
	}
	if d := after.ProjectCount - before.ProjectCount; d != 4 {
		t.Errorf("project count +%d, want 4", d)
	}
	if d := after.FilesBytes - before.FilesBytes; d != 510 {
		t.Errorf("files bytes +%d, want 510", d)
	}
	if d := after.OwnersWithData - before.OwnersWithData; d != 3 {
		t.Errorf("owners +%d, want 3", d)
	}

	top, err := g.TopOwners(1000)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][2]int64{}
	order := []string{}
	for _, o := range top {
		if o.OwnerUserID == a || o.OwnerUserID == b || o.OwnerUserID == c {
			got[o.OwnerUserID] = [2]int64{o.ProjectsBytes, o.FilesBytes}
			order = append(order, o.OwnerUserID)
		}
	}
	if got[a] != [2]int64{150, 10} || got[b] != [2]int64{0, 500} || got[c] != [2]int64{30, 0} {
		t.Fatalf("top=%v", got)
	}
	if len(order) != 3 || order[0] != b || order[1] != a || order[2] != c {
		t.Fatalf("order=%v, want b (500), a (160), c (30)", order)
	}

	items, err := g.ListProjectsForOwner(a, 100)
	if err != nil || len(items) != 3 {
		t.Fatalf("projects of a: %d err=%v (deleted one must be hidden)", len(items), err)
	}
	if n, err := g.ProjectsDatabaseBytes(); err != nil || n <= 0 {
		t.Fatalf("db size=%d err=%v", n, err)
	}
}
