package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// minSigningKeyLen guards against empty or guessable HMAC keys. The access key signs BFF
// sessions, legacy JWTs and play tokens: with a weak key anyone can forge a SuperAdmin session.
const minSigningKeyLen = 32

// ValidateSecrets fails startup when signing keys are missing or too short.
func ValidateSecrets() error {
	for key, env := range map[string]string{
		"auth.access_signing_key":  "AUTH_ACCESS_SIGNING_KEY",
		"auth.refresh_signing_key": "AUTH_REFRESH_SIGNING_KEY",
	} {
		if len(strings.TrimSpace(viper.GetString(key))) < minSigningKeyLen {
			return fmt.Errorf("%s (env %s) must be at least %d characters", key, env, minSigningKeyLen)
		}
	}
	return nil
}
