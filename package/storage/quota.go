// Package storage is the account storage quota shared by everything a user keeps in the ЛК:
// Scratch projects (projectPage) and their own files (userfiles). One limit, one sum.
package storage

import (
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
)

// ErrStorageQuotaExceeded — the account total is used up; only growth is refused.
var ErrStorageQuotaExceeded = errors.New("STORAGE_QUOTA_EXCEEDED")

// UsageSource is one kind of stored data, summed per owner in bytes.
type UsageSource interface {
	GetTotalStorageBytesForOwner(ownerUserID string) (int64, error)
}

// Usage is what the account stores, by kind.
type Usage struct {
	Projects int64
	Files    int64
}

func (u Usage) Total() int64 { return u.Projects + u.Files }

// Quota resolves the account limit from its licence and checks growth against the sum of
// every source. A nil source counts as zero.
type Quota struct {
	licensing licensing.Gateway
	projects  UsageSource
	files     UsageSource
}

func NewQuota(licensingGateway licensing.Gateway, projects, files UsageSource) *Quota {
	return &Quota{licensing: licensingGateway, projects: projects, files: files}
}

func (q *Quota) Usage(ownerUserID string) (Usage, error) {
	var u Usage
	var err error
	if q.projects != nil {
		if u.Projects, err = q.projects.GetTotalStorageBytesForOwner(ownerUserID); err != nil {
			return Usage{}, err
		}
	}
	if q.files != nil {
		if u.Files, err = q.files.GetTotalStorageBytesForOwner(ownerUserID); err != nil {
			return Usage{}, err
		}
	}
	return u, nil
}

// LimitBytes is the account total in bytes; 0 means unlimited (also without a licensing gateway).
func (q *Quota) LimitBytes(ownerUserID string) (int64, error) {
	if q == nil || q.licensing == nil {
		return 0, nil
	}
	ent, err := licensing.ResolveEntitlements(q.licensing, ownerUserID)
	if err != nil {
		return 0, err
	}
	if ent.StorageQuotaMB <= 0 {
		return 0, nil
	}
	return int64(ent.StorageQuotaMB) << 20, nil
}

// AllowNewItem refuses a new item of unknown size (a new, empty project) once the quota is
// used up.
func (q *Quota) AllowNewItem(ownerUserID string) error {
	limit, err := q.LimitBytes(ownerUserID)
	if err != nil || limit == 0 {
		return err
	}
	u, err := q.Usage(ownerUserID)
	if err != nil {
		return err
	}
	if u.Total() >= limit {
		return ErrStorageQuotaExceeded
	}
	return nil
}

// AllowGrowth refuses only growth past the account total: an item that keeps or shrinks its
// size always passes, even over the quota (e.g. after a paid licence expired), so nothing is
// ever lost. A new file is currentSize 0.
func (q *Quota) AllowGrowth(ownerUserID string, currentSize, newSize int64) error {
	if q == nil || newSize <= currentSize {
		return nil
	}
	limit, err := q.LimitBytes(ownerUserID)
	if err != nil || limit == 0 {
		return err
	}
	u, err := q.Usage(ownerUserID)
	if err != nil {
		return err
	}
	if u.Total()-currentSize+newSize > limit {
		return ErrStorageQuotaExceeded
	}
	return nil
}
