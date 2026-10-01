package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestValidateSecrets(t *testing.T) {
	strong := strings.Repeat("k", minSigningKeyLen)
	t.Cleanup(func() {
		viper.Set("auth.access_signing_key", "")
		viper.Set("auth.refresh_signing_key", "")
	})
	viper.Set("auth.access_signing_key", strong)
	viper.Set("auth.refresh_signing_key", strong)
	if err := ValidateSecrets(); err != nil {
		t.Fatalf("strong keys rejected: %v", err)
	}
	for _, weak := range []string{"", "short", strings.Repeat(" ", 40)} {
		viper.Set("auth.access_signing_key", weak)
		if err := ValidateSecrets(); err == nil {
			t.Fatalf("access key %q accepted", weak)
		}
	}
}
