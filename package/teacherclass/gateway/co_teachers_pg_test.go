//go:build integration

package gateway

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

func coTeacherTestClass(t *testing.T, db *gorm.DB) *models.ClassInviteDB {
	t.Helper()
	inv := &models.ClassInviteDB{ID: uuid.NewString(), OwnerTeacherID: "t-" + uuid.NewString()[:8], DisplayName: "c",
		InviteCode: uuid.NewString()[:8], InviteSlug: uuid.NewString()[:8], IsActive: true}
	inv.EdxCohortID = models.UnlinkedCohortKey(inv.ID)
	if err := db.Create(inv).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_invites WHERE id = ?`, inv.ID) })
	return inv
}

func TestCoTeacherTerm(t *testing.T) {
	g, db := fileTestGateway(t)
	inv := coTeacherTestClass(t, db)
	teacher := "c-" + uuid.NewString()[:8]
	term := func(id string, until time.Time) *models.ClassCoTeacherDB {
		return &models.ClassCoTeacherDB{ClassID: inv.ID, TeacherID: id, GrantedBy: inv.OwnerTeacherID, ExpiresAt: until}
	}
	day := 24 * time.Hour

	if err := g.AddCoTeacher(term(teacher, time.Now().Add(10*day)), 2); err != nil {
		t.Fatal(err)
	}
	if err := g.AddCoTeacher(term(teacher, time.Now().Add(day)), 2); !errors.Is(err, teacherclass.ErrAlreadyCoTeacher) {
		t.Fatalf("twice: %v", err)
	}
	other := "c-" + uuid.NewString()[:8]
	if err := g.AddCoTeacher(term(other, time.Now().Add(day)), 2); err != nil {
		t.Fatal(err)
	}
	if err := g.AddCoTeacher(term("c-third", time.Now().Add(day)), 2); !errors.Is(err, teacherclass.ErrTooManyCoTeachers) {
		t.Fatalf("over the limit: %v", err)
	}

	if ok, err := g.IsCoTeacher(inv.ID, teacher); err != nil || !ok {
		t.Fatalf("IsCoTeacher = %v, %v", ok, err)
	}
	mine, err := g.ListInvitesByTeacher(teacher)
	if err != nil || len(mine) != 1 || mine[0].ID != inv.ID {
		t.Fatalf("ListInvitesByTeacher = %+v, %v", mine, err)
	}

	// Ends within a day: claimed once, whoever asks next gets nothing.
	claimed, err := g.ClaimExpiringCoTeachers(time.Now().Add(2 * day))
	if err != nil || len(claimed) != 1 || claimed[0].TeacherID != other {
		t.Fatalf("claimed = %+v, %v", claimed, err)
	}
	if again, _ := g.ClaimExpiringCoTeachers(time.Now().Add(2 * day)); len(again) != 0 {
		t.Fatalf("claimed twice: %+v", again)
	}

	// Extending returns the row and allows a new reminder.
	row, err := g.SetCoTeacherExpiry(inv.ID, other, time.Now().Add(5*day), nil)
	if err != nil || row.TeacherID != other || row.RemindedAt != nil {
		t.Fatalf("extended = %+v, %v", row, err)
	}
	if _, err := g.SetCoTeacherExpiry(inv.ID, "c-none", time.Now().Add(day), nil); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("no term: %v", err)
	}

	if err := g.RevokeCoTeacher(inv.ID, other, inv.OwnerTeacherID); err != nil {
		t.Fatal(err)
	}
	if ok, _ := g.IsCoTeacher(inv.ID, other); ok {
		t.Fatal("revoked term still counts")
	}
	if err := g.RevokeCoTeacher(inv.ID, other, inv.OwnerTeacherID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("revoked twice: %v", err)
	}
	if was, _ := g.WasCoTeacher(inv.ID, other); !was {
		t.Fatal("a past term is remembered")
	}

	// An ended term stops counting by itself and does not block a new one.
	db.Exec(`UPDATE class_co_teachers SET expires_at = now() - interval '1 minute' WHERE class_id = ? AND teacher_id = ?`, inv.ID, teacher)
	if ok, _ := g.IsCoTeacher(inv.ID, teacher); ok {
		t.Fatal("ended term still counts")
	}
	if mine, _ := g.ListInvitesByTeacher(teacher); len(mine) != 0 {
		t.Fatalf("ended term lists the class: %+v", mine)
	}
	if err := g.AddCoTeacher(term(teacher, time.Now().Add(day)), 2); err != nil {
		t.Fatalf("new term after an ended one: %v", err)
	}
	var terms int64
	db.Model(&models.ClassCoTeacherDB{}).Where("class_id = ? AND teacher_id = ?", inv.ID, teacher).Count(&terms)
	if terms != 2 {
		t.Fatalf("history keeps both terms: %d", terms)
	}

	// An archived class takes its terms with it.
	db.Exec(`UPDATE class_invites SET archived_at = now() WHERE id = ?`, inv.ID)
	if list, _ := g.ListCoTeachings(teacher); len(list) != 0 {
		t.Fatalf("archived class still co-taught: %+v", list)
	}
	if err := g.AddCoTeacher(term("c-late", time.Now().Add(day)), 2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("archived class: %v", err)
	}
}

func TestReviewerRecordedAndClearedByHandIn(t *testing.T) {
	g, db := fileTestGateway(t)
	inv := coTeacherTestClass(t, db)
	p := &models.ScratchProjectDB{OwnerUserID: "s", Title: "w", ScratchVMJSON: "{}", EdxCohortID: &inv.EdxCohortID}
	if err := db.Omit("Tags").Create(p).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM scratch_projects WHERE id = ?`, p.ID) })

	if err := g.UpdateReview(p.ID, models.ReviewAccepted, "ok", "teacher-1", nil); err != nil {
		t.Fatal(err)
	}
	got, err := g.GetProject(p.ID)
	if err != nil || got.ReviewedByTeacherID == nil || *got.ReviewedByTeacherID != "teacher-1" || got.ReviewedAt == nil {
		t.Fatalf("reviewed = %+v, %v", got, err)
	}
	handIn := &teacherclass.HandIn{At: time.Now().UTC()}
	if err := g.UpdateReview(p.ID, models.ReviewSubmitted, "", "", handIn); err != nil {
		t.Fatal(err)
	}
	got, err = g.GetProject(p.ID)
	if err != nil || got.ReviewedByTeacherID != nil || got.ReviewedAt != nil {
		t.Fatalf("handed in again = %+v, %v", got, err)
	}
}
