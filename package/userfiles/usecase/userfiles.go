package usecase

import (
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/blobstore"
	"github.com/spf13/viper"
)

const (
	defaultMaxUploadMB          = 50
	defaultMaxFilesPerUser      = 1000
	defaultRetainDeletedDays    = 7
	defaultMaxConcurrentUploads = 2
	defaultMaxUploadsPerHour    = 60
	defaultMinFreeDiskMB        = 1024
	gcBatch                     = 500
	// An upload still in tmp after this long was cut off.
	staleUploadAge = time.Hour
)

// Config is the userFiles section of config.yml.
type Config struct {
	Enabled           bool
	MaxUploadMB       int64
	MaxFilesPerUser   int64
	RetainDeletedDays int
	// Per account: uploads at the same time, and uploads per hour (deleted ones included).
	MaxConcurrentUploads int
	MaxUploadsPerHour    int64
	// Uploads stop while the blob volume has less free space than this plus one file.
	MinFreeDiskMB int64
}

func ConfigFromViper() Config {
	c := Config{
		Enabled:           viper.GetBool("userFiles.enabled"),
		MaxUploadMB:       viper.GetInt64("userFiles.maxUploadMB"),
		MaxFilesPerUser:   viper.GetInt64("userFiles.maxFilesPerUser"),
		RetainDeletedDays: viper.GetInt("userFiles.retainDeletedDays"),

		MaxConcurrentUploads: viper.GetInt("userFiles.maxConcurrentUploads"),
		MaxUploadsPerHour:    viper.GetInt64("userFiles.maxUploadsPerHour"),
		MinFreeDiskMB:        viper.GetInt64("userFiles.minFreeDiskMB"),
	}
	if c.MaxUploadMB <= 0 {
		c.MaxUploadMB = defaultMaxUploadMB
	}
	if c.MaxFilesPerUser <= 0 {
		c.MaxFilesPerUser = defaultMaxFilesPerUser
	}
	if c.RetainDeletedDays <= 0 {
		c.RetainDeletedDays = defaultRetainDeletedDays
	}
	if c.MaxConcurrentUploads <= 0 {
		c.MaxConcurrentUploads = defaultMaxConcurrentUploads
	}
	if c.MaxUploadsPerHour <= 0 {
		c.MaxUploadsPerHour = defaultMaxUploadsPerHour
	}
	if !viper.IsSet("userFiles.minFreeDiskMB") {
		c.MinFreeDiskMB = defaultMinFreeDiskMB
	}
	return c
}

type UserFilesUseCaseImpl struct {
	cfg              Config
	gateway          userfiles.Gateway
	blobs            blobstore.BlobStore
	quota            *storage.Quota
	licensingGateway licensing.Gateway

	mu       sync.Mutex
	inFlight map[string]int // uploads in progress per owner (one backend instance)
}

func NewUserFilesUseCase(
	cfg Config,
	gateway userfiles.Gateway,
	blobs blobstore.BlobStore,
	quota *storage.Quota,
	licensingGateway licensing.Gateway,
) *UserFilesUseCaseImpl {
	return &UserFilesUseCaseImpl{
		cfg:              cfg,
		gateway:          gateway,
		blobs:            blobs,
		quota:            quota,
		licensingGateway: licensingGateway,
		inFlight:         map[string]int{},
	}
}

// acquireSlot limits uploads in progress per account: each one holds up to a full file on
// disk before the final quota check, so parallel requests could otherwise fill the volume.
func (u *UserFilesUseCaseImpl) acquireSlot(ownerUserID string) (release func(), err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.inFlight[ownerUserID] >= u.cfg.MaxConcurrentUploads {
		return nil, userfiles.ErrTooManyUploads
	}
	u.inFlight[ownerUserID]++
	return func() {
		u.mu.Lock()
		defer u.mu.Unlock()
		if u.inFlight[ownerUserID]--; u.inFlight[ownerUserID] <= 0 {
			delete(u.inFlight, ownerUserID)
		}
	}, nil
}

func (u *UserFilesUseCaseImpl) checkFreeSpace(need int64) error {
	free, err := u.blobs.FreeBytes()
	if err != nil {
		log.Printf("userfiles: free space: %v", err)
		return userfiles.ErrStorageUnavailable
	}
	if free >= 0 && free < need+u.cfg.MinFreeDiskMB<<20 {
		log.Printf("userfiles: uploads paused, %d MB free on the blob volume", free>>20)
		return userfiles.ErrStorageUnavailable
	}
	return nil
}

func (u *UserFilesUseCaseImpl) Enabled() bool { return u.cfg.Enabled && u.blobs != nil }

// MaxUploadBytes is the largest file the owner may upload: the tariff's per-project size
// (10 MB Free / 50 MB paid), capped by userFiles.maxUploadMB.
func (u *UserFilesUseCaseImpl) MaxUploadBytes(ownerUserID string) (int64, error) {
	ceiling := u.cfg.MaxUploadMB << 20
	if u.licensingGateway == nil {
		return ceiling, nil
	}
	ent, err := licensing.ResolveEntitlements(u.licensingGateway, ownerUserID)
	if err != nil {
		return 0, err
	}
	if ent.MaxProjectSizeMB <= 0 {
		return ceiling, nil
	}
	return min(int64(ent.MaxProjectSizeMB)<<20, ceiling), nil
}

func (u *UserFilesUseCaseImpl) checkRoom(ownerUserID string, size int64) error {
	recent, err := u.gateway.CountCreatedSince(ownerUserID, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= u.cfg.MaxUploadsPerHour {
		return userfiles.ErrUploadRateLimited
	}
	n, err := u.gateway.CountForOwner(ownerUserID)
	if err != nil {
		return err
	}
	if n >= u.cfg.MaxFilesPerUser {
		return userfiles.ErrFileCountExceeded
	}
	return u.quota.AllowGrowth(ownerUserID, 0, size)
}

// inspectContent refuses files of an allowed type whose contents can run code when opened,
// and reports Python scripts inside an archive (allowed, but the ЛК warns about them).
func inspectContent(kind string, ra io.ReaderAt, size int64) (hasScripts bool, err error) {
	switch kind {
	case kindZIP, kindDOCX, kindXLSX, kindPPTX, kindODT, kindSB3, kindSB2, kindFCSTD:
		return inspectZip(kind, ra, size)
	case kindSKP:
		// SketchUp 2021+ saves a zip; older versions their own binary stream.
		magic := make([]byte, 4)
		if _, err := ra.ReadAt(magic, 0); err == nil && string(magic) == "PK\x03\x04" {
			return inspectZip(kind, ra, size)
		}
	case kindPDF:
		return false, checkPDF(ra, size)
	}
	return false, nil
}

// sanitizeImage stores a re-encoded copy instead of the uploaded image (see imagecheck.go).
func (u *UserFilesUseCaseImpl) sanitizeImage(kind string, src blobstore.Staged, maxBytes int64) (blobstore.Staged, error) {
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(reencodeImage(kind, src, src.Size(), pw))
	}()
	clean, err := u.blobs.Stage(pr, maxBytes)
	pr.Close() // unblocks the encoder when Stage stopped early
	if errors.Is(err, blobstore.ErrTooLarge) {
		return nil, userfiles.ErrFileSizeExceeded
	}
	return clean, err
}

func withExt(name, kind string) string {
	if kind != kindPNG || strings.EqualFold(filepath.Ext(name), ".png") {
		return name
	}
	return strings.TrimSuffix(name, filepath.Ext(name)) + ".png"
}

func (u *UserFilesUseCaseImpl) Upload(ownerUserID, name string, r io.Reader) (*models.UserFileDB, error) {
	if !u.Enabled() {
		return nil, userfiles.ErrDisabled
	}
	name = SanitizeName(name)
	if kindForName(name) == "" {
		return nil, userfiles.ErrFileTypeNotAllowed
	}
	release, err := u.acquireSlot(ownerUserID)
	if err != nil {
		return nil, err
	}
	defer release()
	// Refuse a full account before reading the body; the exact check runs under the lock.
	if err := u.checkRoom(ownerUserID, 1); err != nil {
		return nil, err
	}
	maxBytes, err := u.MaxUploadBytes(ownerUserID)
	if err != nil {
		return nil, err
	}
	if err := u.checkFreeSpace(maxBytes); err != nil {
		return nil, err
	}
	staged, err := u.blobs.Stage(r, maxBytes)
	if errors.Is(err, blobstore.ErrTooLarge) {
		return nil, userfiles.ErrFileSizeExceeded
	}
	if err != nil {
		return nil, err
	}
	defer staged.Discard()
	kind, mime, err := Classify(name, staged.Head(), staged, staged.Size())
	if err != nil {
		return nil, err
	}
	hasScripts, err := inspectContent(kind, staged, staged.Size())
	if err != nil {
		return nil, err
	}
	if isImageKind(kind) {
		clean, err := u.sanitizeImage(kind, staged, maxBytes)
		if err != nil {
			return nil, err
		}
		defer clean.Discard()
		staged = clean
		kind = storedImageKind(kind)
		mime = mimeByKind[kind]
		name = withExt(name, kind)
	}
	row := &models.UserFileDB{
		ID:           uuid.NewString(),
		OwnerUserID:  ownerUserID,
		OriginalName: name,
		Kind:         kind,
		Mime:         mime,
		SizeBytes:    staged.Size(),
		SHA256:       staged.SHA256(),
		HasScripts:   hasScripts,
		CreatedAt:    time.Now().UTC(),
	}
	if err := u.blobs.Publish(staged, row.ID); err != nil {
		return nil, err
	}
	if err := u.gateway.CreateLocked(row, func() error { return u.checkRoom(ownerUserID, row.SizeBytes) }); err != nil {
		if delErr := u.blobs.Delete(row.ID); delErr != nil {
			log.Printf("userfiles: remove blob %s after a refused upload: %v", row.ID, delErr)
		}
		return nil, err
	}
	return row, nil
}

func (u *UserFilesUseCaseImpl) List(ownerUserID string) ([]*models.UserFileDB, error) {
	if !u.Enabled() {
		return nil, userfiles.ErrDisabled
	}
	return u.gateway.ListByOwner(ownerUserID)
}

func (u *UserFilesUseCaseImpl) CountFiles(ownerUserID string) (int64, error) {
	if !u.Enabled() {
		return 0, nil
	}
	return u.gateway.CountForOwner(ownerUserID)
}

func (u *UserFilesUseCaseImpl) Open(ownerUserID, id string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	if !u.Enabled() {
		return nil, nil, userfiles.ErrDisabled
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil, userfiles.ErrFileNotFound
	}
	row, err := u.gateway.GetByOwnerAndID(ownerUserID, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := u.blobs.Open(row.ID)
	if errors.Is(err, blobstore.ErrNotFound) {
		log.Printf("userfiles: blob of file %s is missing", row.ID)
		return nil, nil, userfiles.ErrFileNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return row, rc, nil
}

// Delete frees the quota and the disk at once: the bytes go now, the row stays for
// userFiles.retainDeletedDays as a record. Keeping the bytes too would let an account fill
// the volume by uploading and deleting in a loop.
func (u *UserFilesUseCaseImpl) Delete(ownerUserID, id string) error {
	if !u.Enabled() {
		return userfiles.ErrDisabled
	}
	if _, err := uuid.Parse(id); err != nil {
		return userfiles.ErrFileNotFound
	}
	if err := u.gateway.SoftDelete(ownerUserID, id); err != nil {
		return err
	}
	if err := u.blobs.Delete(id); err != nil {
		// The GC tries again with the row.
		log.Printf("userfiles: delete blob %s: %v", id, err)
	}
	return nil
}

// CollectGarbage removes old rows of deleted files, and their bytes if Delete could not.
// The blob goes first, so a failure never leaves bytes without a row.
func (u *UserFilesUseCaseImpl) CollectGarbage(now time.Time) (int, error) {
	if u.blobs == nil {
		return 0, nil
	}
	removed := 0
	cutoff := now.Add(-time.Duration(u.cfg.RetainDeletedDays) * 24 * time.Hour)
	for {
		ids, err := u.gateway.ListDeletedBefore(cutoff, gcBatch)
		if err != nil {
			return removed, err
		}
		if len(ids) == 0 {
			break
		}
		done := make([]string, 0, len(ids))
		for _, id := range ids {
			if err := u.blobs.Delete(id); err != nil {
				log.Printf("userfiles: delete blob %s: %v", id, err)
				continue
			}
			done = append(done, id)
		}
		if err := u.gateway.HardDelete(done); err != nil {
			return removed, err
		}
		removed += len(done)
		if len(ids) < gcBatch || len(done) == 0 {
			break
		}
	}
	if _, err := u.blobs.SweepTemp(now.Add(-staleUploadAge)); err != nil {
		return removed, err
	}
	return removed, nil
}
