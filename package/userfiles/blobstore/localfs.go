package blobstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	headSize = 512
	tmpDir   = "tmp"
	filesDir = "files"
	filePerm = 0o640
	dirPerm  = 0o750
)

// Keys are file ids (UUIDs); nothing else may reach the file system.
var keyPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// LocalFS stores blobs under root: root/tmp for uploads in progress and
// root/files/<first two chars>/<key> for published ones. Both are on one file system, so
// Publish is a rename.
type LocalFS struct {
	root string
}

// NewLocalFS creates the directories and checks that root is writable.
func NewLocalFS(root string) (*LocalFS, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("blobstore: storage directory is not set")
	}
	for _, d := range []string{filepath.Join(root, tmpDir), filepath.Join(root, filesDir)} {
		if err := os.MkdirAll(d, dirPerm); err != nil {
			return nil, fmt.Errorf("blobstore: %w", err)
		}
	}
	probe, err := os.CreateTemp(filepath.Join(root, tmpDir), "probe-*")
	if err != nil {
		return nil, fmt.Errorf("blobstore: %s is not writable: %w", root, err)
	}
	probe.Close()
	_ = os.Remove(probe.Name())
	return &LocalFS{root: root}, nil
}

func (s *LocalFS) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.root, filesDir, key[:2], key), nil
}

type headWriter struct{ buf []byte }

func (w *headWriter) Write(p []byte) (int, error) {
	if room := headSize - len(w.buf); room > 0 {
		w.buf = append(w.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

type localStaged struct {
	f         *os.File
	sum       string
	size      int64
	head      []byte
	published bool
}

func (s *localStaged) ReadAt(p []byte, off int64) (int, error) { return s.f.ReadAt(p, off) }
func (s *localStaged) SHA256() string                          { return s.sum }
func (s *localStaged) Size() int64                             { return s.size }
func (s *localStaged) Head() []byte                            { return s.head }

func (s *localStaged) Discard() error {
	if s.published {
		return nil
	}
	s.f.Close()
	if err := os.Remove(s.f.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalFS) Stage(r io.Reader, maxBytes int64) (Staged, error) {
	f, err := os.CreateTemp(filepath.Join(s.root, tmpDir), "upload-*")
	if err != nil {
		return nil, err
	}
	staged := &localStaged{f: f}
	fail := func(err error) (Staged, error) {
		_ = staged.Discard()
		return nil, err
	}
	if err := f.Chmod(filePerm); err != nil {
		return fail(err)
	}
	hasher := sha256.New()
	head := &headWriter{}
	n, err := io.Copy(io.MultiWriter(f, hasher, head), io.LimitReader(r, maxBytes+1))
	if err != nil {
		return fail(err)
	}
	if n > maxBytes {
		return fail(ErrTooLarge)
	}
	if err := f.Sync(); err != nil {
		return fail(err)
	}
	staged.sum = hex.EncodeToString(hasher.Sum(nil))
	staged.size = n
	staged.head = head.buf
	return staged, nil
}

func (s *LocalFS) Publish(st Staged, key string) error {
	staged, ok := st.(*localStaged)
	if !ok || staged.published {
		return errors.New("blobstore: not a staged LocalFS upload")
	}
	dst, err := s.path(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	if err := staged.f.Close(); err != nil {
		return err
	}
	if err := os.Rename(staged.f.Name(), dst); err != nil {
		return err
	}
	staged.published = true
	// The rename itself must survive a power cut, not only the data.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

func (s *LocalFS) Open(key string) (io.ReadSeekCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *LocalFS) Delete(key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalFS) SweepTemp(olderThan time.Time) (int, error) {
	dir := filepath.Join(s.root, tmpDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || !info.ModTime().Before(olderThan) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}
