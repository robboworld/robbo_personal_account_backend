package profanity

import (
	"strings"
	"testing"
)

func TestContainsProfanity_Direct(t *testing.T) {
	cases := []string{
		"это хуй",
		"блять",
		"what the fuck",
		"asshole",
	}
	for _, c := range cases {
		if !ContainsProfanity(c) {
			t.Errorf("expected profanity in %q", c)
		}
	}
}

func TestContainsProfanity_Bypasses(t *testing.T) {
	cases := []string{
		"х у й",
		"х.у.й",
		"бл@дь",
		"xyй",
		"f.u.c.k",
	}
	for _, c := range cases {
		if !ContainsProfanity(c) {
			t.Errorf("expected bypass detection for %q", c)
		}
	}
}

func TestContainsProfanity_Clean(t *testing.T) {
	cases := []string{
		"классный проект",
		"привет, как дела?",
		"great scratch project",
		"",
		"hello world",
	}
	for _, c := range cases {
		if ContainsProfanity(c) {
			t.Errorf("unexpected profanity in %q", c)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("Х.У.Й"); got != "хуй" {
		t.Errorf("Normalize = %q, want хуй", got)
	}
	if got := Normalize("ёлка"); !strings.Contains(got, "елка") {
		t.Errorf("Normalize ёлка = %q", got)
	}
}
