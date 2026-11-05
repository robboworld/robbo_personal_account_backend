//go:build integration

package gateway

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// A new hand-in ends the teacher's edit: the marks go, and the kept version is left to retention.
func TestHandInClearsTeacherEdit(t *testing.T) {
	g, db := fileTestGateway(t)
	id, version := uuid.NewString(), uuid.NewString()
	if err := db.Exec(`INSERT INTO scratch_projects (id, owner_user_id, title) VALUES (?, 'student', 'work')`, id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, id) })
	sum := sha256.Sum256([]byte("student"))
	if err := db.Exec(`INSERT INTO scratch_project_versions
		(id, project_id, version_seq, archive, size_bytes, checksum_sha256, created_by_user_id)
		VALUES (?, ?, 1, 'student', 7, ?, 'student')`, version, id, sum[:]).Error; err != nil {
		t.Fatal(err)
	}
	mark := func() {
		t.Helper()
		if err := db.Exec(`UPDATE scratch_projects
			SET teacher_edited_by = 'teacher', teacher_edited_at = now(), student_version_id = ? WHERE id = ?`, version, id).Error; err != nil {
			t.Fatal(err)
		}
	}
	read := func() models.ScratchProjectDB {
		t.Helper()
		var row models.ScratchProjectDB
		if err := db.Where("id = ?", id).First(&row).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}

	mark()
	// A review is not a hand-in: the edit stays visible when the work goes back to the student.
	if err := g.UpdateReview(id, models.ReviewReturned, "Доработай", "teacher", nil); err != nil {
		t.Fatal(err)
	}
	if row := read(); row.TeacherEditedBy == nil || row.TeacherEditedAt == nil || row.StudentVersionID == nil {
		t.Fatalf("review dropped the edit: %+v", row)
	}

	handIn := &teacherclass.HandIn{At: time.Now().UTC()}
	if err := g.UpdateReview(id, models.ReviewSubmitted, "", "", handIn); err != nil {
		t.Fatal(err)
	}
	if row := read(); row.TeacherEditedBy != nil || row.TeacherEditedAt != nil || row.StudentVersionID != nil {
		t.Fatalf("hand-in kept the edit: %+v", row)
	}
}

func TestListActiveEditorsTakesOnlyFreshSignals(t *testing.T) {
	g, db := fileTestGateway(t)
	id := uuid.NewString()
	if err := db.Exec(`INSERT INTO scratch_projects (id, owner_user_id, title) VALUES (?, 'student', 'work')`, id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, id) })
	if err := db.Exec(`INSERT INTO scratch_project_editors (project_id, user_id, role, last_seen_at) VALUES
		(?, 'teacher', 'teacher', now()),
		(?, 'gone', 'teacher', now() - make_interval(secs => ?))`, id, id, models.EditingTTLSeconds+1).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := g.ListActiveEditors([]string{id})
	if err != nil || len(rows) != 1 || rows[0].UserID != "teacher" {
		t.Fatalf("rows = %+v, %v", rows, err)
	}
	if rows, err = g.ListActiveEditors(nil); err != nil || len(rows) != 0 {
		t.Fatalf("no ids: %+v, %v", rows, err)
	}
}
