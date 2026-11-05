//go:build integration

package gateway

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"gorm.io/gorm"
)

// handedInWork is an assignment copy on review with the student's save as its current version.
func handedInWork(t *testing.T, db *gorm.DB, status string) string {
	t.Helper()
	id := templateProject(t, db, false, 1, "student")
	assignment := uuid.NewString()
	if err := db.Exec(`INSERT INTO class_assignments
		(id, edx_course_id, edx_cohort_id, created_by_teacher_id, title, assignment_key)
		VALUES (?, 'c', ?, 'teacher', 'Лабиринт', 'k')`, assignment, assignment).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, assignment) })
	if err := db.Exec(`UPDATE scratch_projects SET owner_user_id = 'student', review_status = ?,
		assignment_id = ?, version_counter = 1, preview_image = 'x', preview_mime = 'image/png'
		WHERE id = ?`, status, assignment, id).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func workRow(t *testing.T, db *gorm.DB, id string) models.ScratchProjectDB {
	t.Helper()
	var row models.ScratchProjectDB
	if err := db.Where("id = ?", id).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

// keepOnlyLatest is the default retention: no history, like Scratch.
var keepOnlyLatest = projectPage.Sb3RetentionPolicy{KeepLast: 1}

func TestTeacherEditKeepsStudentVersionThroughRetention(t *testing.T) {
	g, db := templateTestGateway(t)
	id := handedInWork(t, db, models.ReviewSubmitted)
	student := *workRow(t, db, id).CurrentVersionID

	for _, body := range []string{"teacher 1", "teacher 2"} {
		if err := g.SaveTeacherEdit(id, "teacher", []byte(body)); err != nil {
			t.Fatal(err)
		}
		if _, err := g.PruneSb3Versions(id, keepOnlyLatest); err != nil {
			t.Fatal(err)
		}
	}
	row := workRow(t, db, id)
	if row.StudentVersionID == nil || *row.StudentVersionID != student {
		t.Fatalf("student version = %v, want %s", row.StudentVersionID, student)
	}
	if row.TeacherEditedBy == nil || *row.TeacherEditedBy != "teacher" || row.TeacherEditedAt == nil {
		t.Fatalf("marks = %v %v", row.TeacherEditedBy, row.TeacherEditedAt)
	}
	if archive, err := g.GetLatestSb3Archive(id); err != nil || string(archive) != "teacher 2" {
		t.Fatalf("current = %q, %v", archive, err)
	}
	ids, err := g.ListSb3PruneCandidates(keepOnlyLatest, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range ids {
		if candidate == id {
			t.Fatal("the kept version makes the project a prune candidate")
		}
	}
	var versions int64
	db.Model(&models.ScratchProjectVersionDB{}).Where("project_id = ?", id).Count(&versions)
	if versions != 2 {
		t.Fatalf("versions = %d, want the student's and the teacher's last", versions)
	}
}

func TestRestoreStudentVersionUndoesTheEdit(t *testing.T) {
	g, db := templateTestGateway(t)
	id := handedInWork(t, db, models.ReviewAccepted)
	student := *workRow(t, db, id).CurrentVersionID
	if err := g.SaveTeacherEdit(id, "teacher", []byte("teacher")); err != nil {
		t.Fatal(err)
	}

	archive, err := g.RestoreStudentVersion(id)
	if err != nil || string(archive) != "student" {
		t.Fatalf("archive = %q, %v", archive, err)
	}
	row := workRow(t, db, id)
	if *row.CurrentVersionID != student || row.StudentVersionID != nil || row.TeacherEditedBy != nil || row.TeacherEditedAt != nil {
		t.Fatalf("row = %+v", row)
	}
	if len(row.PreviewImage) != 0 {
		t.Fatal("the teacher's preview stayed")
	}
	if current, err := g.GetLatestSb3Archive(id); err != nil || string(current) != "student" {
		t.Fatalf("current = %q, %v", current, err)
	}
	if _, err := g.RestoreStudentVersion(id); !errors.Is(err, projectPage.ErrSb3ArchiveNotFound) {
		t.Fatalf("second restore err = %v", err)
	}
}

func TestTeacherWritesRefusedOnceWorkIsBackWithStudent(t *testing.T) {
	g, db := templateTestGateway(t)
	id := handedInWork(t, db, models.ReviewSubmitted)
	if err := g.SaveTeacherEdit(id, "teacher", []byte("teacher")); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE scratch_projects SET review_status = ? WHERE id = ?`, models.ReviewReturned, id).Error; err != nil {
		t.Fatal(err)
	}

	if err := g.SaveTeacherEdit(id, "teacher", []byte("late")); !errors.Is(err, projectPage.ErrWorkNotWithTeacher) {
		t.Fatalf("save err = %v", err)
	}
	if _, err := g.RestoreStudentVersion(id); !errors.Is(err, projectPage.ErrWorkNotWithTeacher) {
		t.Fatalf("restore err = %v", err)
	}
	if archive, err := g.GetLatestSb3Archive(id); err != nil || string(archive) != "teacher" {
		t.Fatalf("current = %q, %v", archive, err)
	}
}

func TestEditorSignalExpiresAndLeavesTheProjectAlone(t *testing.T) {
	g, db := templateTestGateway(t)
	id := handedInWork(t, db, models.ReviewSubmitted)
	before := workRow(t, db, id).UpdatedAt

	for i := 0; i < 2; i++ {
		if err := g.TouchProjectEditor(id, "teacher", models.EditorTeacher); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := g.ListProjectEditors(id)
	if err != nil || len(rows) != 1 || rows[0].UserID != "teacher" {
		t.Fatalf("rows = %+v, %v", rows, err)
	}
	// The board shows updated_at as the last save.
	if after := workRow(t, db, id).UpdatedAt; !after.Equal(before) {
		t.Fatalf("updated_at moved: %v -> %v", before, after)
	}

	if err := db.Exec(`UPDATE scratch_project_editors SET last_seen_at = now() - make_interval(secs => ?) WHERE project_id = ?`,
		models.EditingTTLSeconds+1, id).Error; err != nil {
		t.Fatal(err)
	}
	if rows, err = g.ListProjectEditors(id); err != nil || len(rows) != 0 {
		t.Fatalf("expired rows = %+v, %v", rows, err)
	}
}
