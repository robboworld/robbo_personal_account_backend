package oidc

import "testing"

func TestLogoutSourceAllowed(t *testing.T) {
	t.Setenv("LMS_URL", "http://lms.test")
	t.Setenv("LMS_PUBLIC_URL", "https://edu.example.org")
	t.Setenv("ROBBO_RS_URL", "http://rs.test")
	const self = "http://lk-api.test"
	cases := []struct {
		name, origin, referer string
		want                  bool
	}{
		{"no headers (LMS redirect/iframe, typed URL)", "", "", true},
		{"own frontend", "http://localhost:3030", "", true},
		{"LMS referer", "", "https://edu.example.org/logout", true},
		{"LMS MFE subdomain", "", "https://apps.edu.example.org/learning/course", true},
		{"RS origin", "http://rs.test", "", true},
		{"backend itself", "", self + "/home", true},
		{"foreign origin", "https://evil.example", "", false},
		{"foreign referer", "", "https://evil.example/page", false},
		{"suffix lookalike", "", "https://evil-edu.example.org/", false},
		{"null origin", "null", "", false},
		{"origin wins over allowed referer", "https://evil.example", "http://localhost:3030/", false},
		{"malformed referer", "", "::::", false},
	}
	for _, tc := range cases {
		if got := LogoutSourceAllowed(tc.origin, tc.referer, self); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
