package lmsdb

import "testing"

func TestParseAvatarIDFromMeta(t *testing.T) {
	cases := []struct {
		meta string
		want string
	}{
		{"", ""},
		{"{}", ""},
		{`{"lk_avatar_id":"ava2"}`, "ava2"},
		{`{"lk_avatar_id":"ava99"}`, ""},
		{`{"marketing_emails_opt_in":true,"lk_avatar_id":"ava3"}`, "ava3"},
		{"not-json", ""},
	}
	for _, tc := range cases {
		got := ParseAvatarIDFromMeta(tc.meta)
		if got != tc.want {
			t.Fatalf("ParseAvatarIDFromMeta(%q)=%q want %q", tc.meta, got, tc.want)
		}
	}
}

func TestSetAvatarIDInMeta(t *testing.T) {
	merged, err := SetAvatarIDInMeta(`{"company":"Robbo"}`, "ava1")
	if err != nil {
		t.Fatal(err)
	}
	if ParseAvatarIDFromMeta(merged) != "ava1" {
		t.Fatalf("expected ava1 in %s", merged)
	}
	cleared, err := SetAvatarIDInMeta(merged, "")
	if err != nil {
		t.Fatal(err)
	}
	if ParseAvatarIDFromMeta(cleared) != "" {
		t.Fatalf("expected cleared avatar in %s", cleared)
	}
	if _, err := SetAvatarIDInMeta("{}", "ava99"); err == nil {
		t.Fatal("expected invalid avatar error")
	}
}
