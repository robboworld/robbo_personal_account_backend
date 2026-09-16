package avatars

import (
	"errors"
	"strings"
)

const (
	Ava1 = "ava1"
	Ava2 = "ava2"
)

// ValidIDs is the whitelist of static avatar ids shipped with the frontend.
var ValidIDs = map[string]struct{}{
	Ava1: {},
	Ava2: {},
}

// ErrInvalidAvatarID is returned when avatarId is not empty and not in ValidIDs.
var ErrInvalidAvatarID = errors.New("invalid avatar id")

// IsValidAvatarID reports whether id is one of ava1 or ava2.
func IsValidAvatarID(id string) bool {
	_, ok := ValidIDs[strings.TrimSpace(id)]
	return ok
}

// NormalizeAvatarID returns trimmed id, or "" for clear/unset.
// Non-empty invalid ids return ErrInvalidAvatarID.
func NormalizeAvatarID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil
	}
	if !IsValidAvatarID(id) {
		return "", ErrInvalidAvatarID
	}
	return id, nil
}
