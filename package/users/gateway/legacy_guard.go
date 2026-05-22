package gateway

import (
	"errors"

	"github.com/spf13/viper"
)

var ErrLegacyUsersDisabled = errors.New("legacy user store disabled: use OIDC sign-in")

func legacyUsersEnabled() bool {
	return viper.GetBool("legacyPostgres.enabled")
}
