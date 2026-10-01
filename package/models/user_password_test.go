package models

import "testing"

func TestUserHTTPFromCoreHidesPassword(t *testing.T) {
	var out StudentHTTP
	out.UserHTTP = &UserHTTP{}
	out.FromCore(&StudentCore{UserCore: UserCore{Id: "1", Email: "s@example.com", Password: "sha1-hash"}})
	if out.UserHTTP.Password != "" {
		t.Fatalf("password exposed: %q", out.UserHTTP.Password)
	}
}
