// Package blobstore keeps the bytes of user files. The ЛК uses LocalFS (a volume on VPS-2);
// an S3 implementation can replace it behind the same interface.
package blobstore

import (
	"errors"
	"io"
	"time"
)

var (
	ErrTooLarge   = errors.New("blob is larger than allowed")
	ErrNotFound   = errors.New("blob not found")
	ErrInvalidKey = errors.New("invalid blob key")
)

// Staged is an upload written to temporary storage, not yet visible under a key.
// It can be read back (ReadAt) to check its contents before Publish.
type Staged interface {
	io.ReaderAt
	SHA256() string
	Size() int64
	// Head is up to the first 512 bytes, for content sniffing.
	Head() []byte
	// Discard removes the staged data; a no-op after Publish.
	Discard() error
}

type BlobStore interface {
	// Stage streams r to temporary storage while hashing it; ErrTooLarge past maxBytes.
	Stage(r io.Reader, maxBytes int64) (Staged, error)
	// Publish makes the staged data readable under key, atomically.
	Publish(s Staged, key string) error
	Open(key string) (io.ReadSeekCloser, error)
	// Delete removes the blob; a missing one is not an error.
	Delete(key string) error
	// SweepTemp removes staged data older than olderThan (uploads cut off by a crash).
	SweepTemp(olderThan time.Time) (removed int, err error)
	// FreeBytes is the space left for new blobs; -1 when the store cannot tell.
	FreeBytes() (int64, error)
}
