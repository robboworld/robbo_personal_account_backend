//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	TEACHERCLASS_TEST_DSN="host=... dbname=..." go test -tags integration ./package/teacherclass/gateway/
package gateway

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// The copy gets the template's current .sb3 even when a newer version exists: saving content
// identical to an older version makes that older version current again.
func TestForkProjectForStudentCopiesCurrentSb3(t *testing.T) {
	dsn := os.Getenv("TEACHERCLASS_TEST_DSN")
	if dsn == "" {
		t.Skip("TEACHERCLASS_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	g := &GatewayImpl{db: db}

	tplID := uuid.NewString()
	if err := db.Exec(`INSERT INTO scratch_projects (id, owner_user_id, title, scratch_vm_json)
		VALUES (?, 'teacher', 'template', '{"t":1}')`, tplID).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, tplID) })
	versionIDs := []string{uuid.NewString(), uuid.NewString()}
	for i, body := range []string{"current", "newer"} {
		sum := sha256.Sum256([]byte(body))
		if err := db.Exec(`INSERT INTO scratch_project_versions
			(id, project_id, version_seq, archive, size_bytes, checksum_sha256, created_by_user_id)
			VALUES (?, ?, ?, ?, ?, ?, 'teacher')`, versionIDs[i], tplID, i+1, []byte(body), len(body), sum[:]).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Exec(`UPDATE scratch_projects SET current_version_id = ?, version_counter = 2 WHERE id = ?`, versionIDs[0], tplID)

	assignment := &models.ClassAssignmentDB{EdxCourseID: "course", EdxCohortID: "cohort",
		CreatedByTeacherID: "teacher", Title: "a", AssignmentKey: uuid.NewString(), TemplateProjectID: &tplID}
	if err := db.Create(assignment).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, assignment.ID) })

	var template models.ScratchProjectDB
	if err := db.Where("id = ?", tplID).First(&template).Error; err != nil {
		t.Fatal(err)
	}
	copyRow, err := g.ForkProjectForStudent(&template, "student", "a", assignment)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, copyRow.ID) })

	var got models.ScratchProjectVersionDB
	if err := db.Where("id = ?", *copyRow.CurrentVersionID).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if string(got.Archive) != "current" || got.VersionSeq != 1 || got.ProjectID != copyRow.ID {
		t.Errorf("copy's version: archive %q, seq %d; want the template's current archive as version 1",
			got.Archive, got.VersionSeq)
	}
}
