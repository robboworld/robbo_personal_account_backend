package oidc

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// PKCEStore persists Authorization Code PKCE state across BFF replicas.
type PKCEStore interface {
	Save(state string, entry PKCEEntry) error
	Peek(state string) (PKCEEntry, bool)
	Consume(state string) (PKCEEntry, bool)
}

type memoryPKCEStore struct {
	mu    sync.Mutex
	items map[string]PKCEEntry
}

func newMemoryPKCEStore() *memoryPKCEStore {
	return &memoryPKCEStore{items: map[string]PKCEEntry{}}
}

func (s *memoryPKCEStore) Save(state string, entry PKCEEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.ExpiresAt = time.Now().Add(10 * time.Minute)
	s.items[state] = entry
	now := time.Now()
	for k, v := range s.items {
		if now.After(v.ExpiresAt) {
			delete(s.items, k)
		}
	}
	return nil
}

func (s *memoryPKCEStore) Peek(state string) (PKCEEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[state]
	if !ok || time.Now().After(e.ExpiresAt) {
		return PKCEEntry{}, false
	}
	return e, true
}

func (s *memoryPKCEStore) Consume(state string) (PKCEEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[state]
	if !ok || time.Now().After(e.ExpiresAt) {
		return PKCEEntry{}, false
	}
	delete(s.items, state)
	return e, true
}

var (
	storeMu      sync.RWMutex
	defaultStore PKCEStore = newMemoryPKCEStore()
)

func currentStore() PKCEStore {
	storeMu.RLock()
	defer storeMu.RUnlock()
	return defaultStore
}

func SetPKCEStore(s PKCEStore) {
	if s == nil {
		return
	}
	storeMu.Lock()
	defaultStore = s
	storeMu.Unlock()
}

// InitSharedStoreFromConfig uses Licensing Postgres when DSN is set.
func InitSharedStoreFromConfig() error {
	dsn := viper.GetString("licensingPostgres.postgresDsn")
	if dsn == "" {
		dsn = viper.GetString("LICENSING_POSTGRES_DSN")
	}
	if dsn == "" {
		log.Printf("[oidc] PKCE store: memory (no LICENSING_POSTGRES_DSN)")
		return nil
	}
	pg, err := newPostgresPKCEStore(dsn)
	if err != nil {
		return err
	}
	SetPKCEStore(pg)
	log.Printf("[oidc] PKCE store: licensing postgres")
	return nil
}
