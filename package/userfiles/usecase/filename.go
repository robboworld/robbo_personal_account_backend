package usecase

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxNameRunes = 255

// SanitizeName keeps only the last path segment, drops control and bidi-override characters
// (they disguise "exe.txt" as "txt.exe") and caps the length, keeping the extension.
func SanitizeName(name string) string {
	name = strings.ToValidUTF8(name, "")
	name = strings.ReplaceAll(name, `\`, "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || isBidiControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	if utf8.RuneCountInString(name) > maxNameRunes {
		ext := filepath.Ext(name)
		if utf8.RuneCountInString(ext) > 16 {
			ext = ""
		}
		base := []rune(strings.TrimSuffix(name, ext))
		name = string(base[:maxNameRunes-utf8.RuneCountInString(ext)]) + ext
	}
	return name
}

func isBidiControl(r rune) bool {
	return r >= 0x202A && r <= 0x202E || r >= 0x2066 && r <= 0x2069 || r == 0x200E || r == 0x200F || r == 0x061C
}
