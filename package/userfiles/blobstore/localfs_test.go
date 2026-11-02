package blobstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const key = "0123abcd-0000-4000-8000-000000000001"

func newFS(t *testing.T) *LocalFS {
	t.Helper()
	fs, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func tmpCount(t *testing.T, fs *LocalFS) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(fs.root, tmpDir))
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestStagePublishOpen(t *testing.T) {
	fs := newFS(t)
	data := bytes.Repeat([]byte("abc"), 1000)
	st, err := fs.Stage(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if st.SHA256() != hex.EncodeToString(sum[:]) || st.Size() != int64(len(data)) || len(st.Head()) != headSize {
		t.Fatalf("sha=%s size=%d head=%d", st.SHA256(), st.Size(), len(st.Head()))
	}
	buf := make([]byte, 3)
	if _, err := st.ReadAt(buf, 3); err != nil || string(buf) != "abc" {
		t.Fatalf("ReadAt before publish: %q %v", buf, err)
	}
	if err := fs.Publish(st, key); err != nil {
		t.Fatal(err)
	}
	if err := st.Discard(); err != nil {
		t.Fatalf("Discard after Publish: %v", err)
	}
	rc, err := fs.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatal("published bytes differ")
	}
	if tmpCount(t, fs) != 0 {
		t.Fatal("tmp is not empty after publish")
	}
	if err := fs.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Open(key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("open after delete: %v", err)
	}
	if err := fs.Delete(key); err != nil {
		t.Fatalf("deleting a missing blob: %v", err)
	}
}

func TestStageTooLargeLeavesNothing(t *testing.T) {
	fs := newFS(t)
	if _, err := fs.Stage(bytes.NewReader(make([]byte, 101)), 100); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err=%v, want ErrTooLarge", err)
	}
	if tmpCount(t, fs) != 0 {
		t.Fatal("an oversized upload was left in tmp")
	}
}

func TestDiscardRemovesStaged(t *testing.T) {
	fs := newFS(t)
	st, err := fs.Stage(bytes.NewReader([]byte("x")), 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Discard(); err != nil {
		t.Fatal(err)
	}
	if tmpCount(t, fs) != 0 {
		t.Fatal("discarded upload is still in tmp")
	}
}

func TestKeysCannotEscapeRoot(t *testing.T) {
	fs := newFS(t)
	for _, bad := range []string{"../../etc/passwd", "", "ABCD", key + "/x"} {
		if _, err := fs.Open(bad); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("key %q: err=%v", bad, err)
		}
	}
}

func TestSweepTempKeepsFreshUploads(t *testing.T) {
	fs := newFS(t)
	old := filepath.Join(fs.root, tmpDir, "upload-old")
	fresh := filepath.Join(fs.root, tmpDir, "upload-fresh")
	os.WriteFile(old, []byte("x"), 0o640)
	os.WriteFile(fresh, []byte("x"), 0o640)
	past := time.Now().Add(-2 * time.Hour)
	os.Chtimes(old, past, past)
	n, err := fs.SweepTemp(time.Now().Add(-time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("removed=%d err=%v", n, err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("a fresh upload was swept")
	}
}
