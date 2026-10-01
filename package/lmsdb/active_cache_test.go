package lmsdb

import (
	"errors"
	"testing"
)

func stubLookup(t *testing.T, fn func(int64) (bool, error)) {
	t.Helper()
	orig := lookupUserActive
	lookupUserActive = fn
	t.Cleanup(func() {
		lookupUserActive = orig
		activeCacheMu.Lock()
		activeCache = map[int64]activeCacheEntry{}
		activeCacheMu.Unlock()
	})
}

func TestIsUserActiveCachedErrorIsNotInactiveAndNotCached(t *testing.T) {
	calls := 0
	stubLookup(t, func(int64) (bool, error) {
		calls++
		if calls == 1 {
			return false, errors.New("mysql: connection refused")
		}
		return true, nil
	})
	if _, err := IsUserActiveCached("7"); err == nil {
		t.Fatal("lookup error swallowed")
	}
	active, err := IsUserActiveCached("7")
	if err != nil || !active {
		t.Fatalf("after recovery active=%v err=%v, want active (error must not be cached)", active, err)
	}
}

func TestIsUserActiveCachedCachesResult(t *testing.T) {
	calls := 0
	stubLookup(t, func(int64) (bool, error) { calls++; return false, nil })
	for i := 0; i < 3; i++ {
		if active, err := IsUserActiveCached("8"); err != nil || active {
			t.Fatalf("active=%v err=%v, want inactive", active, err)
		}
	}
	if calls != 1 {
		t.Fatalf("lookups=%d want 1 (cached)", calls)
	}
}

func TestIsUserActiveCachedInvalidID(t *testing.T) {
	stubLookup(t, func(int64) (bool, error) { t.Fatal("lookup for invalid id"); return false, nil })
	if active, err := IsUserActiveCached("alice"); active || err != nil {
		t.Fatalf("active=%v err=%v, want inactive without error", active, err)
	}
}
