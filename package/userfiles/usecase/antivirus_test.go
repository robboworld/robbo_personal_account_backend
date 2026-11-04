package usecase

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/antivirus"
)

type fakeScanner struct {
	sig   string
	err   error
	calls int
}

func (s *fakeScanner) Scan(_ context.Context, r io.Reader) (string, error) {
	s.calls++
	_, _ = io.Copy(io.Discard, r)
	return s.sig, s.err
}

func (s *fakeScanner) Ping(context.Context) error { return s.err }

func withScanner(uc *UserFilesUseCaseImpl, s antivirus.Scanner, required bool) {
	uc.scanner = s
	uc.cfg.Antivirus = antivirus.Config{Address: "clamd:3310", Timeout: time.Second, Required: required}
}

func TestUploadScannedClean(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	sc := &fakeScanner{}
	withScanner(uc, sc, true)
	row, err := uc.Upload("1", "sketch.ino", text(10))
	if err != nil {
		t.Fatal(err)
	}
	if sc.calls != 1 || row.ScannedAt == nil {
		t.Fatalf("calls=%d scannedAt=%v, want one scan recorded", sc.calls, row.ScannedAt)
	}
	if !uc.ExchangeEnabled() {
		t.Fatal("a required scanner should enable file exchange")
	}
}

func TestUploadInfectedRefused(t *testing.T) {
	uc, gw := newUseCase(t, 0)
	withScanner(uc, &fakeScanner{sig: "Win.Test.EICAR_HDB-1"}, false)
	if _, err := uc.Upload("1", "notes.txt", text(10)); !errors.Is(err, userfiles.ErrFileInfected) {
		t.Fatalf("err = %v, want ErrFileInfected", err)
	}
	if n, _ := gw.CountForOwner("1"); n != 0 {
		t.Fatalf("%d rows stored for an infected file", n)
	}
}

func TestUploadScannerDown(t *testing.T) {
	down := &fakeScanner{err: antivirus.ErrUnavailable}

	uc, _ := newUseCase(t, 0)
	withScanner(uc, down, true)
	if _, err := uc.Upload("1", "notes.txt", text(10)); !errors.Is(err, userfiles.ErrScanUnavailable) {
		t.Fatalf("required: err = %v, want ErrScanUnavailable", err)
	}

	uc, _ = newUseCase(t, 0)
	withScanner(uc, down, false)
	row, err := uc.Upload("1", "notes.txt", text(10))
	if err != nil {
		t.Fatalf("not required: %v", err)
	}
	if row.ScannedAt != nil {
		t.Fatal("a skipped scan must not be recorded")
	}
	if uc.ExchangeEnabled() {
		t.Fatal("file exchange needs a required scan")
	}
}

func TestUploadWithoutScanner(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	row, err := uc.Upload("1", "notes.txt", text(10))
	if err != nil || row.ScannedAt != nil {
		t.Fatalf("row=%+v err=%v; want stored unscanned", row, err)
	}
	uc.cfg.Antivirus.Required = true
	if _, err := uc.Upload("1", "notes.txt", text(10)); !errors.Is(err, userfiles.ErrScanUnavailable) {
		t.Fatalf("required without clamd: err = %v", err)
	}
}

func TestEnsureScannedOldFile(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	// Uploaded before the scanner was configured.
	row, err := uc.Upload("1", "notes.txt", text(10))
	if err != nil {
		t.Fatal(err)
	}
	sc := &fakeScanner{}
	withScanner(uc, sc, true)
	got, err := uc.EnsureScanned("1", row.ID)
	if err != nil || got.ScannedAt == nil || sc.calls != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, sc.calls)
	}
	if _, err := uc.EnsureScanned("1", row.ID); err != nil || sc.calls != 1 {
		t.Fatalf("second call scanned again (calls=%d, err=%v)", sc.calls, err)
	}
	if _, err := uc.EnsureScanned("2", row.ID); !errors.Is(err, userfiles.ErrFileNotFound) {
		t.Fatalf("someone else's file: err = %v", err)
	}

	bad, _ := uc.Upload("1", "other.txt", text(5))
	bad.ScannedAt = nil // the stored row, as if it was never scanned
	withScanner(uc, &fakeScanner{sig: "Eicar"}, true)
	if _, err := uc.EnsureScanned("1", bad.ID); !errors.Is(err, userfiles.ErrFileInfected) {
		t.Fatalf("infected old file: err = %v", err)
	}
}

func TestDeleteFileInUse(t *testing.T) {
	uc, _ := newUseCase(t, 0)
	row, err := uc.Upload("1", "notes.txt", text(10))
	if err != nil {
		t.Fatal(err)
	}
	prev := userfiles.FileInUse
	t.Cleanup(func() { userfiles.FileInUse = prev })
	userfiles.FileInUse = func(owner, id string) (bool, error) { return owner == "1" && id == row.ID, nil }
	if err := uc.Delete("1", row.ID); !errors.Is(err, userfiles.ErrFileInUse) {
		t.Fatalf("err = %v, want ErrFileInUse", err)
	}
	userfiles.FileInUse = func(string, string) (bool, error) { return false, nil }
	if err := uc.Delete("1", row.ID); err != nil {
		t.Fatalf("free file: %v", err)
	}
}
