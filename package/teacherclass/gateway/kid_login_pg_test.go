//go:build integration

package gateway

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

func TestKidLoginRoomAndLock(t *testing.T) {
	g, db := fileTestGateway(t)
	inv := coTeacherTestClass(t, db)

	if _, err := g.GetKidRoom(inv.ID); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("no room yet: %v", err)
	}
	slug := uuid.NewString()[:16]
	room := &models.ClassKidLoginRoomDB{ClassID: inv.ID, LoginSlug: slug}
	if err := g.SaveKidRoom(room); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	room.OpenUntil = &until
	room.LoginSlug = uuid.NewString()[:16]
	if err := g.SaveKidRoom(room); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := g.GetKidRoomBySlug(room.LoginSlug)
	if err != nil || got.OpenUntil == nil || !got.OpenUntil.Equal(until) {
		t.Fatalf("room = %+v, %v", got, err)
	}
	if _, err := g.GetKidRoomBySlug(slug); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("old slug: %v", err)
	}

	// Room codes are unique across classes; the second class gets ErrConflict.
	code := "473915"
	room.RoomCode = &code
	if err := g.SaveKidRoom(room); err != nil {
		t.Fatal(err)
	}
	if byCode, err := g.GetKidRoomByCode(code); err != nil || byCode.ClassID != inv.ID {
		t.Fatalf("by code: %+v, %v", byCode, err)
	}
	other := coTeacherTestClass(t, db)
	clash := &models.ClassKidLoginRoomDB{ClassID: other.ID, LoginSlug: uuid.NewString()[:16], RoomCode: &code}
	if err := g.SaveKidRoom(clash); !errors.Is(err, teacherclass.ErrConflict) {
		t.Fatalf("taken code: %v", err)
	}
	notDigits := "12ab56"
	clash.RoomCode = &notDigits
	if err := g.SaveKidRoom(clash); err == nil {
		t.Fatal("letters in a room code must be refused by the CHECK")
	}
	room.RoomCode = nil
	if err := g.SaveKidRoom(room); err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetKidRoomByCode(code); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("cleared code: %v", err)
	}

	kid := &models.ClassKidLoginDB{UserID: "9" + uuid.NewString()[:6], ClassID: inv.ID, DisplayName: "Маша К.",
		Avatar: 3, Pictures: []int64{1, 7, 7, 0, 8}, CreatedBy: inv.OwnerTeacherID}
	if err := g.CreateKidLogin(kid); err != nil {
		t.Fatal(err)
	}
	bad := &models.ClassKidLoginDB{UserID: "8" + uuid.NewString()[:6], ClassID: inv.ID, DisplayName: "x",
		Pictures: []int64{1, 2}, CreatedBy: inv.OwnerTeacherID}
	if err := g.CreateKidLogin(bad); err == nil {
		t.Fatal("two pictures must be refused by the CHECK")
	}

	// Parallel wrong pairs all count; the lock comes at the fifth.
	var wg sync.WaitGroup
	lockedN := 0
	var mu sync.Mutex
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locked, err := g.RecordKidLoginFailure(kid.UserID, 5)
			if err != nil {
				t.Error(err)
				return
			}
			if locked {
				mu.Lock()
				lockedN++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	row, err := g.GetKidLogin(kid.UserID)
	if err != nil || row.FailedAttempts != 6 || row.LockedAt == nil || lockedN != 2 {
		t.Fatalf("after 6 wrong: attempts=%d locked=%v lockedAnswers=%d err=%v", row.FailedAttempts, row.LockedAt, lockedN, err)
	}
	if _, err := g.RecordKidLoginFailure("nobody", 5); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("unknown kid: %v", err)
	}

	if err := g.SetKidPictures(kid.UserID, []int{4, 2, 2, 8, 0}); err != nil {
		t.Fatal(err)
	}
	row, _ = g.GetKidLogin(kid.UserID)
	if row.LockedAt != nil || row.FailedAttempts != 0 || len(row.Pictures) != 5 || row.Pictures[0] != 4 || row.Pictures[4] != 0 {
		t.Fatalf("after new pictures: %+v", row)
	}

	if err := g.SetKidCardHash(kid.UserID, "hash-"+kid.UserID); err != nil {
		t.Fatal(err)
	}
	if byCard, err := g.GetKidLoginByCardHash("hash-" + kid.UserID); err != nil || byCard.UserID != kid.UserID {
		t.Fatalf("by card: %+v, %v", byCard, err)
	}
	if err := g.RecordKidLoginSuccess(kid.UserID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if n, err := g.CountKidLogins(inv.ID); err != nil || n != 1 {
		t.Fatalf("count = %d, %v", n, err)
	}
}
