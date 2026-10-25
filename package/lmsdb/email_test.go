package lmsdb

import "testing"

func TestSameEmail(t *testing.T) {
	cases := []struct {
		current, requested string
		want               bool
	}{
		{"a@b.ru", "a@b.ru", true},
		{"A@B.ru", " a@b.RU ", true},
		{"a@b.ru", "", true},
		{"a@b.ru", "evil@x.ru", false},
	}
	for _, c := range cases {
		if got := sameEmail(c.current, c.requested); got != c.want {
			t.Errorf("sameEmail(%q, %q) = %v, want %v", c.current, c.requested, got, c.want)
		}
	}
}
