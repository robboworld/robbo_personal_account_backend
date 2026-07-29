package auth

import (
	"errors"
	"time"
)

// AccountInactiveError is returned when LMS is_active=0. Optionally carries
// ban reason/expiry from lk_user_bans for the login UI.
type AccountInactiveError struct {
	Reason      string
	ExpiresAt   *time.Time
	IsPermanent bool
	HasBan      bool
}

func (e *AccountInactiveError) Error() string {
	return ErrUserInactive.Error()
}

func (e *AccountInactiveError) Is(target error) bool {
	return target == ErrUserInactive
}

// NewAccountInactiveError builds an inactive error, optionally with ban details.
func NewAccountInactiveError(reason string, expiresAt *time.Time, hasBan bool) error {
	e := &AccountInactiveError{
		Reason:      reason,
		ExpiresAt:   expiresAt,
		HasBan:      hasBan,
		IsPermanent: hasBan && expiresAt == nil,
	}
	return e
}

// AsAccountInactive unwraps AccountInactiveError if present.
func AsAccountInactive(err error) (*AccountInactiveError, bool) {
	var inactive *AccountInactiveError
	if errors.As(err, &inactive) {
		return inactive, true
	}
	return nil, false
}
