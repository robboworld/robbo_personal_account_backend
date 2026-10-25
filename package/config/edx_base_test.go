package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestApplyEdxAPIBase(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("api_urls.postCohort", "https://edx-test.ru/api/cohorts/v1/courses/")
	t.Setenv("API_URLS_GETUSER", "http://override/me")

	applyEdxAPIBase("http://lms:8000/")

	if got := viper.GetString("api_urls.postCohort"); got != "http://lms:8000/api/cohorts/v1/courses/" {
		t.Fatalf("postCohort = %q", got)
	}
	if got := viper.GetString("api_urls.refreshToken"); got != "http://lms:8000/oauth2/access_token" {
		t.Fatalf("refreshToken = %q", got)
	}
	if viper.IsSet("api_urls.getUser") && viper.GetString("api_urls.getUser") == "http://lms:8000/api/user/v1/me" {
		t.Fatal("an explicit API_URLS_GETUSER must not be replaced by the base")
	}
}

func TestApplyEdxAPIBaseEmptyKeepsConfig(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("api_urls.postCohort", "https://edx-test.ru/api/cohorts/v1/courses/")

	applyEdxAPIBase("  ")

	if got := viper.GetString("api_urls.postCohort"); got != "https://edx-test.ru/api/cohorts/v1/courses/" {
		t.Fatalf("postCohort = %q, want unchanged", got)
	}
}
