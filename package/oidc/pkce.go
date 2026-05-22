package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

type PKCEEntry struct {
	State         string
	Nonce         string
	CodeVerifier  string
	CodeChallenge string
	ReturnTo      string
	ExpiresAt     time.Time
}

type pkceEntry = PKCEEntry

var pkceStore = struct {
	mu    sync.Mutex
	items map[string]pkceEntry
}{
	items: map[string]pkceEntry{},
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func savePKCE(state string, entry pkceEntry) {
	pkceStore.mu.Lock()
	defer pkceStore.mu.Unlock()
	entry.ExpiresAt = time.Now().Add(10 * time.Minute)
	pkceStore.items[state] = entry
	for k, v := range pkceStore.items {
		if time.Now().After(v.ExpiresAt) {
			delete(pkceStore.items, k)
		}
	}
}

func loadPKCE(state string) (pkceEntry, bool) {
	pkceStore.mu.Lock()
	defer pkceStore.mu.Unlock()
	e, ok := pkceStore.items[state]
	if !ok || time.Now().After(e.ExpiresAt) {
		return pkceEntry{}, false
	}
	delete(pkceStore.items, state)
	return e, true
}

func NewPKCEForReturn(returnTo string) (PKCEEntry, error) {
	state, err := randomURLSafe(16)
	if err != nil {
		return PKCEEntry{}, err
	}
	nonce, err := randomURLSafe(16)
	if err != nil {
		return PKCEEntry{}, err
	}
	verifier, err := randomURLSafe(48)
	if err != nil {
		return PKCEEntry{}, err
	}
	entry := PKCEEntry{
		State:         state,
		Nonce:         nonce,
		CodeVerifier:  verifier,
		CodeChallenge: codeChallengeS256(verifier),
		ReturnTo:      returnTo,
	}
	savePKCE(state, entry)
	return entry, nil
}

func ConsumePKCE(state string) (PKCEEntry, bool) {
	return loadPKCE(state)
}
