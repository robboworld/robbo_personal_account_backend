package users

import "errors"

var (
	ErrInternalServerLevel = errors.New("internal server error")
	ErrBadRequest          = errors.New("bad request")
	ErrBadRequestBody      = errors.New("bad request body")
	ErrAlreadyUsedEmail    = errors.New("email is already used by another user")
	// Password accounts in the legacy database are gone: people sign in through LMS (OIDC).
	ErrPasswordAccountsRemoved = errors.New("password accounts are no longer created; users sign in through LMS")
)
