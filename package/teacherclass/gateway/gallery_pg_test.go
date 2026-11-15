//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	TEACHERCLASS_TEST_DSN="host=... dbname=..." go test -tags integration ./package/teacherclass/gateway/
package gateway

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

func TestLessonGallery(t *testing.T) {
	g, db := fileTestGateway(t)
	teacher, student := "t-"+uuid.NewString()[:8], "s-"+uuid.NewString()[:8]

	inv := &models.ClassInviteDB{ID: uuid.NewString(), OwnerTeacherID: teacher, DisplayName: "c",
		InviteCode: uuid.NewString()[:8], InviteSlug: uuid.NewString()[:8], IsActive: true}
	inv.EdxCohortID = models.UnlinkedCohortKey(inv.ID)
	if err := db.Create(inv).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_invites WHERE id = ?`, inv.ID) })
	a := &models.ClassAssignmentDB{EdxCohortID: inv.EdxCohortID, CreatedByTeacherID: teacher, Title: "a",
		AssignmentKey: uuid.NewString()}
	if err := g.CreateAssignment(a); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_assignments WHERE id = ?`, a.ID) })

	f1 := insertUserFile(t, db, student, "one.png", "png")
	f2 := insertUserFile(t, db, student, "two.png", "png")
	f3 := insertUserFile(t, db, student, "three.png", "png")
	tf := insertUserFile(t, db, teacher, "board.jpg", "jpg")

	add := func(owner, file string, limit int) (*models.ClassGalleryItemDB, error) {
		it := &models.ClassGalleryItemDB{AssignmentID: a.ID, UserFileID: file, UploaderUserID: owner}
		_, err := g.AddGalleryItem(it, limit)
		return it, err
	}
	i1, err := add(student, f1, 2)
	if err != nil {
		t.Fatal(err)
	}
	i2, err := add(student, f2, 2)
	if err != nil || i2.Position != i1.Position+1 {
		t.Fatalf("second item = %+v, %v; want the next position", i2, err)
	}
	if _, err := add(student, f3, 2); !errors.Is(err, teacherclass.ErrTooManyGalleryFiles) {
		t.Fatalf("over the limit: err = %v", err)
	}
	again := &models.ClassGalleryItemDB{AssignmentID: a.ID, UserFileID: f1, UploaderUserID: student}
	created, err := g.AddGalleryItem(again, 2)
	if err != nil || created || again.ID != i1.ID {
		t.Fatalf("the same file again = %+v, created %v, %v; want the existing item", again, created, err)
	}
	it, err := add(teacher, tf, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := g.ReorderGallery(a.ID, []string{it.ID, i2.ID, i1.ID}); err != nil {
		t.Fatal(err)
	}
	pinned, caption := true, "best"
	if err := g.UpdateGalleryItem(i1.ID, teacherclass.GalleryItemUpdate{Pinned: &pinned, Caption: &caption}); err != nil {
		t.Fatal(err)
	}
	list, err := g.ListGalleryItems(a.ID)
	if err != nil || len(list) != 3 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[0].ID != i1.ID || list[0].Caption != "best" || list[1].ID != it.ID || list[2].ID != i2.ID {
		t.Fatalf("order = %s %s %s; want pinned, then the new order", list[0].ID, list[1].ID, list[2].ID)
	}

	// A file deleted from "Мои файлы" leaves the gallery and frees a place.
	if err := db.Exec(`UPDATE user_files SET deleted_at = now() WHERE id = ?`, f2).Error; err != nil {
		t.Fatal(err)
	}
	counts, err := g.CountGalleryItems([]string{a.ID})
	if err != nil || counts[a.ID] != 2 {
		t.Fatalf("counts = %v, %v", counts, err)
	}
	if _, err := g.GetGalleryItem(i2.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("item of a deleted file: err = %v", err)
	}
	if _, err := add(student, f3, 2); err != nil {
		t.Fatalf("after the delete: %v", err)
	}

	if err := g.SetGalleryClosed(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := g.GetAssignment(a.ID); got.GalleryClosedAt == nil {
		t.Fatal("gallery not closed")
	}
	if err := g.DeleteGalleryItem(i1.ID); err != nil {
		t.Fatal(err)
	}
	if err := g.DeleteGalleryItem(i1.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}

	// Deleting the lesson takes its gallery; the files stay.
	if err := g.DeleteAssignment(a.ID); err != nil {
		t.Fatal(err)
	}
	var left int64
	db.Raw(`SELECT count(*) FROM class_gallery_items WHERE assignment_id = ?`, a.ID).Scan(&left)
	var files int64
	db.Raw(`SELECT count(*) FROM user_files WHERE id IN ? AND deleted_at IS NULL`, []string{f1, f3, tf}).Scan(&files)
	if left != 0 || files != 3 {
		t.Fatalf("after deleting the assignment: %d items, %d files; want 0 and 3", left, files)
	}
}
