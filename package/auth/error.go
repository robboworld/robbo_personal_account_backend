package auth

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("incorrect password")
	ErrUserInactive        = errors.New("user account is disabled")
	ErrLegacyAuthDisabled = errors.New("legacy sign-in disabled: use /auth/oidc/start")
	ErrSignUpDisabled     = errors.New("sign-up disabled: register via LMS or admin")
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrInvalidTypeClaims  = errors.New("token claims are not of type *StandardClaims")
	ErrTokenNotFound      = errors.New("token not found")
	ErrUserAlreadyExist   = errors.New("user already exist")
	ErrNotAccess          = errors.New("no access")
)
