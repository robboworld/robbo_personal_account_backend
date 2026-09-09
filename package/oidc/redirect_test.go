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
	}
	for _, tc := range cases {
		if got := SanitizeReturnTo(tc.in); got != tc.want {
			t.Fatalf("SanitizeReturnTo(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
