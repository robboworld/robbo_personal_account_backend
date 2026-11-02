package usecase

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"report.pdf":                 "report.pdf",
		`C:\Users\kid\homework.docx`: "homework.docx",
		"../../etc/passwd.txt":       "passwd.txt",
		"  space.png  ":              "space.png",
		"bad\x00\x07name.txt":        "badname.txt",
		"invoice\u202Etxt.exe":       "invoicetxt.exe",
		"Мой проект.sb3":             "Мой проект.sb3",
		"...":                        "",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeNameKeepsExtensionWhenCut(t *testing.T) {
	got := SanitizeName(strings.Repeat("я", 400) + ".pdf")
	if utf8.RuneCountInString(got) != maxNameRunes || !strings.HasSuffix(got, ".pdf") {
		t.Fatalf("len=%d name ends %q", utf8.RuneCountInString(got), got[len(got)-8:])
	}
}
