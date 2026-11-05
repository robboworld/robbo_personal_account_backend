//go:build integration

package gateway

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

func sb3WithProject(t *testing.T, projectJSON string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("project.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(projectJSON)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func insertVersion(t *testing.T, db *gorm.DB, projectID string, seq int, archive []byte, saveSource string) string {
	t.Helper()
	id := uuid.NewString()
	sum := sha256.Sum256(archive)
	if err := db.Exec(`INSERT INTO scratch_project_versions
		(id, project_id, version_seq, archive, size_bytes, checksum_sha256, created_by_user_id, save_source)
		VALUES (?, ?, ?, ?, ?, ?, 'student', NULLIF(?, ''))`,
		id, projectID, seq, archive, len(archive), sum[:], saveSource).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

// Everything a deleted assignment leaves behind, for both kinds of work.
func TestDeleteAssignmentLeavesWorkWithStudents(t *testing.T) {
	g, db := fileTestGateway(t)
	cohort := "delete-" + uuid.NewString()
	inv := models.ClassInviteDB{
		EdxCourseID: "course", EdxCohortID: cohort, OwnerTeacherID: "teacher", DisplayName: "5A",
		InviteCode: uuid.NewString(), InviteSlug: uuid.NewString(), IsActive: true,
	}
	if err := g.CreateInvite(&inv); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_invites WHERE id = ?`, inv.ID) })
	assignment := models.ClassAssignmentDB{
		EdxCourseID: "course", EdxCohortID: cohort, CreatedByTeacherID: "teacher",
		Title: "Лабиринт", AssignmentKey: "maze",
	}
	if err := g.CreateAssignment(&assignment); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, assignment.ID) })

	copyOf := func(owner, status string) string {
		t.Helper()
		id := uuid.NewString()
		if err := db.Exec(`INSERT INTO scratch_projects
			(id, owner_user_id, title, edx_course_id, edx_cohort_id, assignment_id, review_status, review_comment, submission_note, submitted_at)
			VALUES (?, ?, 'Лабиринт', 'course', ?, ?, ?, 'Молодец', 'Готово', now())`,
			id, owner, cohort, assignment.ID, status).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, id) })
		return id
	}
	read := func(id string) models.ScratchProjectDB {
		t.Helper()
		var row models.ScratchProjectDB
		if err := db.Where("id = ?", id).First(&row).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}

	untouched := copyOf("s1", models.ReviewDraft)
	insertVersion(t, db, untouched, 1, []byte("issued"), saveSourceAssignmentCopy)
	drafted := copyOf("s2", models.ReviewDraft)
	insertVersion(t, db, drafted, 1, []byte("issued"), saveSourceAssignmentCopy)
	insertVersion(t, db, drafted, 2, []byte("drafted"), "lk.upload")
	accepted := copyOf("s3", models.ReviewAccepted)

	// A teacher changed work on review: the student's version is kept by student_version_id.
	edited := copyOf("s4", models.ReviewSubmitted)
	studentJSON := `{"targets":[{"name":"student"}]}`
	studentVersion := insertVersion(t, db, edited, 1, sb3WithProject(t, studentJSON), "lk.upload")
	teacherVersion := insertVersion(t, db, edited, 2, sb3WithProject(t, `{"targets":[{"name":"teacher"}]}`), "lk.teacher_edit")
	if err := db.Exec(`UPDATE scratch_projects SET current_version_id = ?, student_version_id = ?,
		teacher_edited_by = 'teacher', teacher_edited_at = now(), scratch_vm_json = '{"targets":[{"name":"teacher"}]}',
		preview_image = 'teacher', preview_mime = 'image/png' WHERE id = ?`, teacherVersion, studentVersion, edited).Error; err != nil {
		t.Fatal(err)
	}

	// The same, but the work came back: what is current may be the student's rework.
	returned := copyOf("s5", models.ReviewReturned)
	oldVersion := insertVersion(t, db, returned, 1, sb3WithProject(t, `{"targets":[]}`), "lk.upload")
	reworkVersion := insertVersion(t, db, returned, 2, sb3WithProject(t, `{"targets":[]}`+" "), "lk.upload")
	if err := db.Exec(`UPDATE scratch_projects SET current_version_id = ?, student_version_id = ?,
		teacher_edited_by = 'teacher', teacher_edited_at = now() WHERE id = ?`, reworkVersion, oldVersion, returned).Error; err != nil {
		t.Fatal(err)
	}

	file := insertUserFile(t, db, "s6", "scheme.ino", "code")
	submission, err := g.EnsureFileSubmission(assignment.ID, "s6")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AttachSubmissionFile(submission.ID, file, 10); err != nil {
		t.Fatal(err)
	}
	if err := g.UpdateFileReview(submission.ID, models.ReviewSubmitted, "", "", nil); err != nil {
		t.Fatal(err)
	}
	if inUse, err := g.FileInUse("s6", file); err != nil || !inUse {
		t.Fatalf("before delete: inUse = %v, %v", inUse, err)
	}

	if err := g.DeleteAssignment(assignment.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := g.GetAssignment(assignment.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("assignment still there: %v", err)
	}
	if err := g.DeleteAssignment(assignment.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second delete: %v", err)
	}

	if read(untouched).DeletedAt == nil {
		t.Error("a copy the student never saved must go")
	}
	for name, id := range map[string]string{"draft": drafted, "accepted": accepted, "edited": edited, "returned": returned} {
		row := read(id)
		if row.DeletedAt != nil {
			t.Errorf("%s: deleted", name)
		}
		if row.AssignmentID != nil || row.EdxCourseID != nil || row.EdxCohortID != nil || row.ReviewStatus != nil ||
			row.ReviewComment != nil || row.SubmittedAt != nil || row.SubmissionNote != "" ||
			row.TeacherEditedBy != nil || row.TeacherEditedAt != nil || row.StudentVersionID != nil {
			t.Errorf("%s: class left on the project: %+v", name, row)
		}
	}

	row := read(edited)
	if row.CurrentVersionID == nil || *row.CurrentVersionID != studentVersion || row.ScratchVMJSON != studentJSON ||
		len(row.PreviewImage) != 0 || row.PreviewMime != "" {
		t.Errorf("edited work is not what the student handed in: %+v", row)
	}
	if row := read(returned); row.CurrentVersionID == nil || *row.CurrentVersionID != reworkVersion {
		t.Errorf("returned work lost the rework: %+v", row)
	}

	files, err := g.GetUserFiles([]string{file})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[file]; !ok {
		t.Error("the student lost the file")
	}
	if _, err := g.GetFileSubmission(submission.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("submission still there: %v", err)
	}
	if inUse, err := g.FileInUse("s6", file); err != nil || inUse {
		t.Errorf("after delete: inUse = %v, %v", inUse, err)
	}

	// The key is free for a new assignment.
	again := models.ClassAssignmentDB{
		EdxCourseID: "course", EdxCohortID: cohort, CreatedByTeacherID: "teacher",
		Title: "Лабиринт", AssignmentKey: "maze",
	}
	if err := g.CreateAssignment(&again); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, again.ID) })
}
