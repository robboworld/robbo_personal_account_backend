//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	TEACHERCLASS_TEST_DSN="host=... dbname=..." go test -tags integration ./package/teacherclass/gateway/
package gateway

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

func fileTestGateway(t *testing.T) (*GatewayImpl, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("TEACHERCLASS_TEST_DSN")
	if dsn == "" {
		t.Skip("TEACHERCLASS_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return &GatewayImpl{db: db}, db
}

func insertUserFile(t *testing.T, db *gorm.DB, owner, name, kind string) string {
	t.Helper()
	id := uuid.NewString()
	if err := db.Exec(`INSERT INTO user_files (id, owner_user_id, original_name, kind, mime, size_bytes, sha256)
		VALUES (?, ?, ?, ?, 'application/octet-stream', 10, ?)`, id, owner, name, kind, strings.Repeat("a", 64)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM user_files WHERE id = ?`, id) })
	return id
}

// A file assignment in a ЛК class: hand-in, the attach limit, and which files are in use.
func TestFileSubmissionLifecycle(t *testing.T) {
	g, db := fileTestGateway(t)
	teacher, student := "t-"+uuid.NewString()[:8], "s-"+uuid.NewString()[:8]

	inv := &models.ClassInviteDB{ID: uuid.NewString(), OwnerTeacherID: teacher, DisplayName: "c",
		InviteCode: uuid.NewString()[:8], InviteSlug: uuid.NewString()[:8], IsActive: true}
	inv.EdxCohortID = models.UnlinkedCohortKey(inv.ID)
	if err := db.Create(inv).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_invites WHERE id = ?`, inv.ID) })

	starter := insertUserFile(t, db, teacher, "base.FCStd", "fcstd")
	a := &models.ClassAssignmentDB{EdxCohortID: inv.EdxCohortID, CreatedByTeacherID: teacher, Title: "a",
		AssignmentKey: uuid.NewString(), Kind: models.AssignmentFile, TemplateFileID: &starter}
	if err := g.CreateAssignment(a); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, a.ID) })
	got, err := g.GetAssignment(a.ID)
	if err != nil || !got.IsFileAssignment() || got.AllowedKinds == nil || len(got.AllowedKinds) != 0 {
		t.Fatalf("stored assignment = %+v, %v", got, err)
	}

	sub, err := g.EnsureFileSubmission(a.ID, student)
	if err != nil {
		t.Fatal(err)
	}
	again, err := g.EnsureFileSubmission(a.ID, student)
	if err != nil || again.ID != sub.ID || sub.ReviewStatus != models.ReviewDraft {
		t.Fatalf("second EnsureFileSubmission = %+v, %v; want the same draft", again, err)
	}

	f1 := insertUserFile(t, db, student, "part.FCStd", "fcstd")
	f2 := insertUserFile(t, db, student, "part.stl", "stl")
	f3 := insertUserFile(t, db, student, "extra.stl", "stl")
	for _, f := range []string{f1, f2, f1} {
		if err := g.AttachSubmissionFile(sub.ID, f, 2); err != nil {
			t.Fatalf("attach %s: %v", f, err)
		}
	}
	if err := g.AttachSubmissionFile(sub.ID, f3, 2); !errors.Is(err, teacherclass.ErrTooManySubmissionFiles) {
		t.Fatalf("third file with a limit of 2: %v", err)
	}
	files, err := g.ListSubmissionFiles([]string{sub.ID})
	if err != nil || len(files[sub.ID]) != 2 || files[sub.ID][0].ID != f1 {
		t.Fatalf("files = %+v, %v", files, err)
	}

	inUse := func(owner, id string) bool {
		t.Helper()
		ok, err := g.FileInUse(owner, id)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	if inUse(student, f1) {
		t.Fatal("a draft's file must stay deletable")
	}
	if !inUse(teacher, starter) {
		t.Fatal("the starter file of an open assignment is in use")
	}

	if err := g.UpdateFileReview(sub.ID, models.ReviewSubmitted, "", nil); err != nil {
		t.Fatal(err)
	}
	if !inUse(student, f1) || inUse(student, f3) || inUse("someone", f1) {
		t.Fatal("only the handed-in files of their owner are in use")
	}
	if err := g.UpdateFileReview(sub.ID, models.ReviewReturned, "again", nil); err != nil {
		t.Fatal(err)
	}
	if inUse(student, f1) {
		t.Fatal("returned work may be changed: its files are free")
	}
	g.UpdateFileReview(sub.ID, models.ReviewAccepted, "", nil)
	db.Exec(`UPDATE class_invites SET archived_at = now() WHERE id = ?`, inv.ID)
	if inUse(student, f1) || inUse(teacher, starter) {
		t.Fatal("an archived class frees its files")
	}

	// A file deleted from "Мои файлы" (moderation) drops out of the submission.
	db.Exec(`UPDATE user_files SET deleted_at = now() WHERE id = ?`, f2)
	files, _ = g.ListSubmissionFiles([]string{sub.ID})
	if len(files[sub.ID]) != 1 {
		t.Fatalf("deleted file still listed: %+v", files[sub.ID])
	}
	if err := g.DetachSubmissionFile(sub.ID, f1); err != nil {
		t.Fatal(err)
	}
	if err := g.DetachSubmissionFile(sub.ID, f1); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("detach twice: %v", err)
	}
	byID, err := g.GetUserFiles([]string{starter, f2})
	if err != nil || len(byID) != 1 {
		t.Fatalf("GetUserFiles = %+v, %v; want only the live starter", byID, err)
	}
}
