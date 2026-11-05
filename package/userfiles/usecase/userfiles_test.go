package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/blobstore"
)

const mb = int64(1) << 20

// memGateway stands in for user_files; mu plays the per-owner advisory lock.
type memGateway struct {
	mu   sync.Mutex
	rows map[string]*models.UserFileDB
	lock sync.Mutex
}

func newMemGateway() *memGateway { return &memGateway{rows: map[string]*models.UserFileDB{}} }

func (g *memGateway) each(owner string, fn func(*models.UserFileDB)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.rows {
		if r.OwnerUserID == owner && r.DeletedAt == nil {
			fn(r)
		}
	}
}

func (g *memGateway) GetTotalStorageBytesForOwner(owner string) (int64, error) {
	var n int64
	g.each(owner, func(r *models.UserFileDB) { n += r.SizeBytes })
	return n, nil
}

func (g *memGateway) CountForOwner(owner string) (int64, error) {
	var n int64
	g.each(owner, func(*models.UserFileDB) { n++ })
	return n, nil
}

func (g *memGateway) CountCreatedSince(owner string, t time.Time) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var n int64
	for _, r := range g.rows {
		if r.OwnerUserID == owner && !r.CreatedAt.Before(t) {
			n++
		}
	}
	return n, nil
}

func (g *memGateway) ListByOwner(owner string) ([]*models.UserFileDB, error) {
	var out []*models.UserFileDB
	g.each(owner, func(r *models.UserFileDB) { out = append(out, r) })
	return out, nil
}

func (g *memGateway) GetByOwnerAndID(owner, id string) (*models.UserFileDB, error) {
	var found *models.UserFileDB
	g.each(owner, func(r *models.UserFileDB) {
		if r.ID == id {
			found = r
		}
	})
	if found == nil {
		return nil, userfiles.ErrFileNotFound
	}
	return found, nil
}

func (g *memGateway) CreateLocked(row *models.UserFileDB, check func() error) error {
	g.lock.Lock()
	defer g.lock.Unlock()
	if err := check(); err != nil {
		return err
	}
	g.mu.Lock()
	g.rows[row.ID] = row
	g.mu.Unlock()
	return nil
}

func (g *memGateway) Rename(owner, id, name string) error {
	r, err := g.GetByOwnerAndID(owner, id)
	if err != nil {
		return err
	}
	g.mu.Lock()
	r.OriginalName = name
	g.mu.Unlock()
	return nil
}

func (g *memGateway) SoftDelete(owner, id string) error {
	r, err := g.GetByOwnerAndID(owner, id)
	if err != nil {
		return err
	}
	g.mu.Lock()
	now := time.Now()
	r.DeletedAt = &now
	g.mu.Unlock()
	return nil
}

func (g *memGateway) MarkScanned(id string, at time.Time) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r, ok := g.rows[id]; ok {
		r.ScannedAt = &at
	}
	return nil
}

func (g *memGateway) ListDeletedBefore(t time.Time, limit int) ([]string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var ids []string
	for id, r := range g.rows {
		if r.DeletedAt != nil && r.DeletedAt.Before(t) && len(ids) < limit {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (g *memGateway) HardDelete(ids []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, id := range ids {
		delete(g.rows, id)
	}
	return nil
}

type freeLicensing struct{ licensing.Gateway }

func (freeLicensing) ListLicensesByUser(string) ([]*models.LicenseCore, error) { return nil, nil }

type fixedUsage int64

func (u fixedUsage) GetTotalStorageBytesForOwner(string) (int64, error) { return int64(u), nil }

func newUseCase(t *testing.T, projectBytes int64) (*UserFilesUseCaseImpl, *memGateway) {
	t.Helper()
	fs, err := blobstore.NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gw := newMemGateway()
	lic := freeLicensing{}
	cfg := Config{
		Enabled: true, MaxUploadMB: 50, MaxFilesPerUser: 3, RetainDeletedDays: 7,
		MaxConcurrentUploads: 2, MaxUploadsPerHour: 60,
	}
	return NewUserFilesUseCase(cfg, gw, fs, storage.NewQuota(lic, fixedUsage(projectBytes), gw), lic), gw
}

func text(size int64) io.Reader { return bytes.NewReader(bytes.Repeat([]byte("a"), int(size))) }

func TestUploadOpenDelete(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	row, err := uc.Upload(context.Background(), "1", "notes.txt", text(100))
	if err != nil {
		t.Fatal(err)
	}
	if row.Kind != kindTXT || row.SizeBytes != 100 || len(row.SHA256) != 64 {
		t.Fatalf("row=%+v", row)
	}
	if _, _, err := uc.Open("2", row.ID); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("another user opened the file: err=%v", err)
	}
	_, rc, err := uc.Open("1", row.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if len(got) != 100 {
		t.Fatalf("read %d bytes", len(got))
	}
	if err := uc.Delete("2", row.ID); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("another user deleted the file: err=%v", err)
	}
	if err := uc.Delete("1", row.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := uc.List("1"); len(list) != 0 {
		t.Fatalf("deleted file still listed")
	}
	// The bytes go at once: upload-and-delete in a loop must not fill the disk.
	if _, err := uc.blobs.Open(row.ID); !errors.Is(err, blobstore.ErrNotFound) {
		t.Fatalf("blob of a deleted file is still on disk: %v", err)
	}
}

func TestRenameKeepsTheExtension(t *testing.T) {
	uc, gw := newUseCase(t, 0)
	row, err := uc.Upload(context.Background(), "1", "notes.txt", text(100))
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := uc.Rename("1", row.ID, `../  Домашка 5.TXT `)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.OriginalName != "Домашка 5.TXT" || renamed.Mime != row.Mime || renamed.Kind != row.Kind {
		t.Fatalf("renamed=%+v", renamed)
	}
	for _, name := range []string{"notes.exe", "notes.html", "notes", "", ".txt", "notes.txt.exe"} {
		if _, err := uc.Rename("1", row.ID, name); !errors.Is(err, userfiles.ErrFileTypeNotAllowed) {
			t.Fatalf("Rename(%q): err=%v", name, err)
		}
	}
	if got, _ := gw.GetByOwnerAndID("1", row.ID); got.OriginalName != "Домашка 5.TXT" {
		t.Fatalf("a refused rename changed the name: %q", got.OriginalName)
	}
}

func TestRenameFindsOnlyOwnLiveFiles(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	row, err := uc.Upload(context.Background(), "1", "notes.txt", text(100))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Rename("2", row.ID, "x.txt"); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("another user renamed the file: err=%v", err)
	}
	if _, err := uc.Rename("1", "not-a-uuid", "x.txt"); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("bad id: err=%v", err)
	}
	if err := uc.Delete("1", row.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Rename("1", row.ID, "x.txt"); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("renamed a deleted file: err=%v", err)
	}
}

func TestUploadFlagsPythonScripts(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	row, err := uc.Upload(context.Background(), "1", "homework.zip", bytes.NewReader(zipOf(t, map[string]string{"main.py": "print(1)"})))
	if err != nil {
		t.Fatal(err)
	}
	if !row.HasScripts || !row.ToHTTP().HasScripts {
		t.Fatalf("zip with main.py is not flagged: %+v", row)
	}
	if row, err = uc.Upload(context.Background(), "1", "notes.txt", text(10)); err != nil || row.HasScripts {
		t.Fatalf("plain text flagged: row=%+v err=%v", row, err)
	}
}

func TestUploadRateLimitCountsDeletedFiles(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	uc.cfg.MaxUploadsPerHour = 3
	for i := 0; i < 3; i++ {
		row, err := uc.Upload(context.Background(), "1", "n.txt", text(1))
		if err != nil {
			t.Fatal(err)
		}
		uc.Delete("1", row.ID)
	}
	if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); !errors.Is(err, userfiles.ErrUploadRateLimited) {
		t.Fatalf("err=%v, want UPLOAD_RATE_LIMITED", err)
	}
}

func TestConcurrentUploadSlots(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	r1, err := uc.acquireSlot("1")
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := uc.acquireSlot("1")
	if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); !errors.Is(err, userfiles.ErrTooManyUploads) {
		t.Fatalf("third upload at once: err=%v", err)
	}
	if _, err := uc.acquireSlot("2"); err != nil {
		t.Fatalf("another account is not limited: %v", err)
	}
	r1()
	r2()
	if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); err != nil {
		t.Fatalf("after the others finished: %v", err)
	}
}

type fullDisk struct{ blobstore.BlobStore }

func (fullDisk) FreeBytes() (int64, error) { return 100 << 20, nil }

func TestUploadStopsOnLowDisk(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	uc.cfg.MinFreeDiskMB = 1024
	uc.blobs = fullDisk{uc.blobs}
	if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); !errors.Is(err, userfiles.ErrStorageUnavailable) {
		t.Fatalf("err=%v, want STORAGE_UNAVAILABLE", err)
	}
}

func TestUploadLimits(t *testing.T) {
	// Free: 10 MB per file, 100 MB per account with the projects.
	uc, _ := newUseCase(t, 95*mb)
	if _, err := uc.Upload(context.Background(), "1", "big.txt", text(10*mb+1)); !errors.Is(err, userfiles.ErrFileSizeExceeded) {
		t.Fatalf("file over the tariff size: err=%v", err)
	}
	if _, err := uc.Upload(context.Background(), "1", "fits.txt", text(5*mb)); err != nil {
		t.Fatalf("file up to exactly the quota: %v", err)
	}
	if _, err := uc.Upload(context.Background(), "1", "one-more.txt", text(1)); !errors.Is(err, storage.ErrStorageQuotaExceeded) {
		t.Fatalf("file with the quota used up: err=%v", err)
	}
	if _, err := uc.Upload(context.Background(), "1", "page.html", text(10)); !errors.Is(err, userfiles.ErrFileTypeNotAllowed) {
		t.Fatalf("html: err=%v", err)
	}
}

func TestUploadCountLimit(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	for i := 0; i < 3; i++ {
		if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := uc.Upload(context.Background(), "1", "n.txt", text(1)); !errors.Is(err, userfiles.ErrFileCountExceeded) {
		t.Fatalf("err=%v, want FILE_COUNT_EXCEEDED", err)
	}
}

func TestParallelUploadsCannotOvershootQuota(t *testing.T) {
	uc, gw := newUseCase(t, 92*mb)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = uc.Upload(context.Background(), "1", "half.txt", text(5*mb))
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else if !errors.Is(err, storage.ErrStorageQuotaExceeded) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if used, _ := gw.GetTotalStorageBytesForOwner("1"); ok != 1 || used != 5*mb {
		t.Fatalf("%d uploads succeeded, %d bytes stored; want exactly one", ok, used)
	}
}

func TestCollectGarbageRemovesOldDeleted(t *testing.T) {
	uc, gw := newUseCase(t, 0)
	row, err := uc.Upload(context.Background(), "1", "notes.txt", text(10))
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Delete("1", row.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := uc.CollectGarbage(time.Now()); n != 0 {
		t.Fatalf("removed %d files still in retention", n)
	}
	if n, err := uc.CollectGarbage(time.Now().Add(8 * 24 * time.Hour)); n != 1 || err != nil {
		t.Fatalf("removed=%d err=%v", n, err)
	}
	if _, err := uc.blobs.Open(row.ID); !errors.Is(err, blobstore.ErrNotFound) {
		t.Fatalf("blob survived: %v", err)
	}
	if len(gw.rows) != 0 {
		t.Fatal("row survived")
	}
}

func TestDisabled(t *testing.T) {
	uc := NewUserFilesUseCase(Config{}, newMemGateway(), nil, nil, nil)
	if _, err := uc.Upload(context.Background(), "1", "a.txt", text(1)); !errors.Is(err, userfiles.ErrDisabled) {
		t.Fatalf("err=%v", err)
	}
}
