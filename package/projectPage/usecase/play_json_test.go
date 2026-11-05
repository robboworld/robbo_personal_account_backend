package usecase

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/playtoken"
	"github.com/spf13/viper"
)

func TestGetProjectJSONByTokenGivesThePlayerALoadableProject(t *testing.T) {
	viper.Set("auth.access_signing_key", "test-signing-key-for-play-token")
	t.Cleanup(func() { viper.Set("auth.access_signing_key", "") })

	const saved = `{"targets":[{"isStage":true}],"meta":{"semver":"3.0.0"}}`
	cases := []struct {
		name, stored, want string
	}{
		{name: "copy issued without a template", stored: "{}", want: emptyProjectJson2},
		{name: "nothing stored", stored: "", want: emptyProjectJson2},
		{name: "saved project", stored: saved, want: saved},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uc, lock := lockUseCase(nil, models.ReviewDraft)
			lock.row.ScratchVMJSON = c.stored
			token, _, err := playtoken.Issue(lock.row.ID, lock.row.ID)
			if err != nil {
				t.Fatal(err)
			}
			got, err := uc.GetProjectJSONByToken(lock.row.ID, token)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("json = %.60q, want %.60q", got, c.want)
			}
		})
	}
}
