package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

type PKCEEntry struct {
	State             string
	Nonce             string
	CodeVerifier      string
	CodeChallenge     string
	ReturnTo          string
	Prompt            string
	KickOtherSessions bool
	ExpiresAt         time.Time
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

func savePKCE(state string, entry PKCEEntry) {
	_ = currentStore().Save(state, entry)
}

func peekPKCE(state string) (PKCEEntry, bool) {
	return currentStore().Peek(state)
}

func loadPKCE(state string) (PKCEEntry, bool) {
	return currentStore().Consume(state)
}

func NewPKCEForReturn(returnTo string) (PKCEEntry, error) {
	return NewPKCEForReturnWithPrompt(returnTo, "")
}

func NewPKCEForReturnWithPrompt(returnTo, prompt string) (PKCEEntry, error) {
	return NewPKCEForReturnWithPromptAndKick(returnTo, prompt, false)
}

func NewPKCEForReturnWithPromptAndKick(returnTo, prompt string, kickOtherSessions bool) (PKCEEntry, error) {
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
		State:             state,
		Nonce:             nonce,
		CodeVerifier:      verifier,
		CodeChallenge:     codeChallengeS256(verifier),
		ReturnTo:          SanitizeReturnTo(returnTo),
		Prompt:            prompt,
		KickOtherSessions: kickOtherSessions,
	}
	if err := currentStore().Save(state, entry); err != nil {
		return PKCEEntry{}, err
	}
	return entry, nil
}

func ConsumePKCE(state string) (PKCEEntry, bool) {
	return loadPKCE(state)
}

func PeekPKCE(state string) (PKCEEntry, bool) {
	return peekPKCE(state)
}
