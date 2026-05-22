package auth

import (
	"strings"

	"github.com/spf13/viper"
)

func LmsPasswordFallbackEnabled() bool {
	if viper.GetBool("auth.lmsPasswordFallback") {
		return true
	}
	return strings.EqualFold(viper.GetString("auth.mode"), "lms_db")
}

func IsOidcBffMode() bool {
	return strings.EqualFold(viper.GetString("auth.mode"), "oidc_bff") || viper.GetBool("oidc.enabled")
}
