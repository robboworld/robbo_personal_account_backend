package lmsdb

import (
	"strconv"
	"sync"
	"time"
)

type activeCacheEntry struct {
	active    bool
	expiresAt time.Time
}

var (
	activeCacheMu sync.RWMutex
	activeCache   = map[int64]activeCacheEntry{}
	activeCacheTTL = 60 * time.Second
)

// InvalidateActiveCache drops a cached is_active value after ban/unban.
func InvalidateActiveCache(userID int64) {
	if userID <= 0 {
		return
	}
	activeCacheMu.Lock()
	delete(activeCache, userID)
	activeCacheMu.Unlock()
}

// lookupUserActive reads auth_user.is_active (replaced in tests).
var lookupUserActive = func(id int64) (bool, error) {
	reader, err := NewReaderFromConfig()
	if err != nil {
		return false, err
	}
	defer reader.Close()
	return reader.IsUserActive(id)
}

// IsUserActiveCached returns LMS is_active with a short in-memory TTL cache.
// A lookup error is returned as an error, not as "inactive": callers must still deny the
// request (fail closed, so banned users are not admitted while LMS is unreachable) but must
// not end the session or tell an active user that the account is disabled.
// Errors are not cached. A non-numeric or empty id is inactive.
func IsUserActiveCached(userIDStr string) (bool, error) {
	id, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || id <= 0 {
		return false, nil
	}
	now := time.Now()
	activeCacheMu.RLock()
	if e, ok := activeCache[id]; ok && now.Before(e.expiresAt) {
		activeCacheMu.RUnlock()
		return e.active, nil
	}
	activeCacheMu.RUnlock()

	active, err := lookupUserActive(id)
	if err != nil {
		return false, err
	}
	activeCacheMu.Lock()
	activeCache[id] = activeCacheEntry{active: active, expiresAt: now.Add(activeCacheTTL)}
	activeCacheMu.Unlock()
	return active, nil
}
