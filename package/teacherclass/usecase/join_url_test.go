package usecase

import (
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
)

// deploy/backend.prod.yml passes the LK origin as FRONTEND_PUBLICURL; the image's config.yml
// has localhost, so the join link depends on this env name reaching frontend.publicUrl.
func TestJoinURLFromEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetDefault("frontend.publicUrl", "http://localhost:3030")
	t.Setenv("FRONTEND_PUBLICURL", "https://lk.example.org/")

	got := (&UseCaseImpl{}).toDTO(&models.ClassInviteDB{InviteSlug: "abc"}).JoinURL
	if got != "https://lk.example.org/join/abc" {
		t.Fatalf("JoinURL = %q", got)
	}
}
