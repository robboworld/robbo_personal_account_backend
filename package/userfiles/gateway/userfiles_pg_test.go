//go:build integration

// Runs against a projects database with the user_files schema:
//
//	USERFILES_TEST_DSN="host=... dbname=..." go test -tags integration ./package/userfiles/gateway/
package gateway

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

var errFull = errors.New("full")

func testGateway(t *testing.T) *UserFilesGatewayImpl {
	t.Helper()
	dsn := os.Getenv("USERFILES_TEST_DSN")
	if dsn == "" {
		t.Skip("USERFILES_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return NewUserFilesGateway(db)
}

func newRow(owner string, size int64) *models.UserFileDB {
	return &models.UserFileDB{
		ID: uuid.NewString(), OwnerUserID: owner, OriginalName: "a.txt", Kind: "txt", Mime: "text/plain",
		SizeBytes: size, SHA256: strings.Repeat("0", 64), CreatedAt: time.Now().UTC(),
	}
}

func TestGatewayRoundTrip(t *testing.T) {
	g := testGateway(t)
	owner := "it-" + uuid.NewString()
	row := newRow(owner, 100)
	if err := g.CreateLocked(row, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if n, _ := g.GetTotalStorageBytesForOwner(owner); n != 100 {
		t.Fatalf("total=%d", n)
	}
	if _, err := g.GetByOwnerAndID("someone-else", row.ID); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("someone else's file: %v", err)
	}
	if err := g.SoftDelete(owner, row.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := g.CountForOwner(owner); n != 0 {
		t.Fatalf("count after delete=%d", n)
	}
	ids, err := g.ListDeletedBefore(time.Now().Add(time.Minute), 10)
	if err != nil || len(ids) == 0 {
		t.Fatalf("deleted ids=%v err=%v", ids, err)
	}
	if err := g.HardDelete([]string{row.ID}); err != nil {
		t.Fatal(err)
	}
}

// Two uploads at once with room for one: the advisory lock lets exactly one in.
func TestCreateLockedSerializesPerOwner(t *testing.T) {
	g := testGateway(t)
	owner := "it-" + uuid.NewString()
	const limit = 150
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = g.CreateLocked(newRow(owner, 100), func() error {
				used, err := g.GetTotalStorageBytesForOwner(owner)
				if err != nil {
					return err
				}
				time.Sleep(50 * time.Millisecond)
				if used+100 > limit {
					return errFull
				}
				return nil
			})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else if !errors.Is(err, errFull) {
			t.Fatal(err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d inserts passed the check, want 1", ok)
	}
}
