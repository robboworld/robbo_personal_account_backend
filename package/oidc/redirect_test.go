package oidc

import "testing"

func TestSanitizeReturnTo(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/home"},
		{"/projects/1", "/projects/1"},
		{"//evil.example/phish", "/home"},
		{"javascript:alert(1)", "/home"},
		{"http://localhost:3030/home", "http://localhost:3030/home"},
		{"http://localhost:8601/", "http://localhost:8601/"},
		{"https://evil.example/steal", "/home"},
		// Backslash / encoded tricks browsers normalise to //evil.com.
		{"/\\evil.com", "/home"},
		{"\\\\evil.com", "/home"},
		{"/%5Cevil.com", "/home"},
		{"/%5cevil.com", "/home"},
		{"/\tevil.com", "/home"},
		{"/projects/1?tab=info", "/projects/1?tab=info"},
		{"http://localhost:3030@evil.example/", "/home"},
		{"https://scratch.example.com/", "/home"},
	}
	for _, tc := range cases {
		if got := SanitizeReturnTo(tc.in); got != tc.want {
			t.Fatalf("SanitizeReturnTo(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeReturnToLmsAlias(t *testing.T) {
	t.Setenv("LMS_URL", "http://local.overhang.io")
	if got := SanitizeReturnTo("lms"); got != "http://local.overhang.io/" {
		t.Fatalf("SanitizeReturnTo(lms)=%q want http://local.overhang.io/", got)
	}
}

func TestProductLanding(t *testing.T) {
	t.Setenv("LMS_URL", "http://local.overhang.io")
	t.Setenv("ROBBO_RS_URL", "http://localhost:8601")
	if got := ProductLanding("lms"); got != "http://local.overhang.io/" {
		t.Fatalf("ProductLanding(lms)=%q", got)
	}
	if got := ProductLanding("rs"); got != "http://localhost:8601/" {
		t.Fatalf("ProductLanding(rs)=%q", got)
	}
	if got := ProductLanding("lk"); got != "http://localhost:3030/" {
		t.Fatalf("ProductLanding(lk)=%q", got)
	}
}
