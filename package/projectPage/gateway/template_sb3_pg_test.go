//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	PROJECTPAGE_TEST_DSN="host=... dbname=..." go test -tags integration ./package/projectPage/gateway/
package gateway

import (
	"crypto/sha256"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"gorm.io/gorm"
)

func templateTestGateway(t *testing.T) (*ProjectPageGatewayImpl, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("PROJECTPAGE_TEST_DSN")
	if dsn == "" {
		t.Skip("PROJECTPAGE_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return &ProjectPageGatewayImpl{projectStorageDB: db}, db
}

// templateProject inserts a project with one version per archive (version_seq 1..n) and makes
// version current (1-based; 0 leaves current_version_id unset).
func templateProject(t *testing.T, db *gorm.DB, deleted bool, current int, archives ...string) string {
	t.Helper()
	id := uuid.NewString()
	if err := db.Exec(`INSERT INTO scratch_projects (id, owner_user_id, title, deleted_at)
		VALUES (?, 'teacher', 'template', CASE WHEN ? THEN now() END)`, id, deleted).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, id) })
	for i, body := range archives {
		vid := uuid.NewString()
		sum := sha256.Sum256([]byte(body))
		if err := db.Exec(`INSERT INTO scratch_project_versions
			(id, project_id, version_seq, archive, size_bytes, checksum_sha256, created_by_user_id)
			VALUES (?, ?, ?, ?, ?, ?, 'teacher')`, vid, id, i+1, []byte(body), len(body), sum[:]).Error; err != nil {
			t.Fatal(err)
		}
		if current == i+1 {
			db.Exec(`UPDATE scratch_projects SET current_version_id = ? WHERE id = ?`, vid, id)
		}
	}
	return id
}

func TestGetTemplateSb3Archive(t *testing.T) {
	gw, db := templateTestGateway(t)
	cases := []struct {
		name string
		id   string
		want string
	}{
		{"current version, not the newest", templateProject(t, db, false, 1, "v1", "v2"), "v1"},
		{"newest when current is unset", templateProject(t, db, false, 0, "v1", "v2"), "v2"},
		{"deleted template", templateProject(t, db, true, 1, "v1"), "v1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := gw.GetTemplateSb3Archive(c.id)
			if err != nil || string(got) != c.want {
				t.Errorf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}

	for name, id := range map[string]string{
		"template without versions": templateProject(t, db, false, 0),
		"unknown template":          uuid.NewString(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := gw.GetTemplateSb3Archive(id); !errors.Is(err, projectPage.ErrSb3ArchiveNotFound) {
				t.Errorf("err = %v, want ErrSb3ArchiveNotFound", err)
			}
		})
	}
}
