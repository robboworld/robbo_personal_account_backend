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

// IsUserActiveCached returns LMS is_active with a short in-memory TTL cache.
// On LMS misconfiguration or lookup errors it returns true (fail-open) so a
// transient MySQL blip does not lock out the whole LK.
func IsUserActiveCached(userIDStr string) bool {
	id, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || id <= 0 {
		return true
	}
	now := time.Now()
	activeCacheMu.RLock()
	if e, ok := activeCache[id]; ok && now.Before(e.expiresAt) {
		activeCacheMu.RUnlock()
		return e.active
	}
	activeCacheMu.RUnlock()

	reader, err := NewReaderFromConfig()
	if err != nil {
		return true
	}
	defer reader.Close()
	active, err := reader.IsUserActive(id)
	if err != nil {
		return true
	}
	activeCacheMu.Lock()
	activeCache[id] = activeCacheEntry{active: active, expiresAt: now.Add(activeCacheTTL)}
	activeCacheMu.Unlock()
	return active
}
