package licensing

import "errors"

var (
	ErrLicenseNotFound      = errors.New("INVALID_KEY")
	ErrSeatNotFound         = errors.New("SEAT_NOT_FOUND")
	ErrSeatLimitReached     = errors.New("SEAT_LIMIT_REACHED")
	ErrSubscriptionExpired  = errors.New("SUBSCRIPTION_EXPIRED")
	ErrReactivationCooldown = errors.New("REACTIVATION_COOLDOWN")
	ErrDeviceFingerprint    = errors.New("DEVICE_FINGERPRINT_REQUIRED")
	ErrForbidden            = errors.New("FORBIDDEN")
	ErrDeviceLinkNotFound   = errors.New("DEVICE_LINK_NOT_FOUND")
	ErrDeviceLinkPending    = errors.New("authorization_pending")
	ErrDeviceLinkExpired    = errors.New("DEVICE_LINK_EXPIRED")
	ErrBadRequest           = errors.New("bad_request")
	ErrAddonAuthRequired    = errors.New("addon_auth_required")
	ErrInvalidToken         = errors.New("invalid_token_shape")
	ErrUnsupportedAlg       = errors.New("unsupported_alg")
	ErrInvalidSignature     = errors.New("invalid_signature")
	ErrLicenseExpiredToken  = errors.New("license_expired")
	ErrDeviceBinding        = errors.New("device_binding_mismatch")
	ErrSeatNotActive        = errors.New("seat_not_active")
)
