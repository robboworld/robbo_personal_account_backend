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
	ErrUserAlreadyExist      = errors.New("user already exist")
	ErrEmailAlreadyExist     = errors.New("email already exists")
	ErrUsernameAlreadyExist  = errors.New("username already exists")
	ErrCompanyRequired       = errors.New("company name is required")
	ErrHonorCodeRequired     = errors.New("honor code consent is required")
	ErrMarketingOptInRequired = errors.New("marketing opt-in consent is required")
	ErrInvalidRegistration   = errors.New("invalid registration data")
	ErrNotAccess          = errors.New("no access")
)
