package profanity

import (
	"strings"
	"unicode"
)

// Normalize prepares text for dictionary matching: lowercasing, ё→е,
// leetspeak, stripping punctuation/spaces, collapsing repeats.
func Normalize(text string) string {
	text = strings.ToLower(text)
	var b strings.Builder
	b.Grow(len(text))
	var prev rune
	for _, r := range text {
		r = mapLeet(r)
		if r == 'ё' {
			r = 'е'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if r == prev {
				continue
			}
			b.WriteRune(r)
			prev = r
			continue
		}
		// Drop punctuation / whitespace — collapses "х у й" / "х.у.й"
		prev = 0
	}
	return b.String()
}

func mapLeet(r rune) rune {
	switch r {
	case '0':
		return 'о'
	case '3':
		return 'з'
	case '4':
		return 'ч'
	case '@':
		return 'я'
	case '$':
		return 'с'
	case 'x', 'X':
		return 'х'
	case 'y', 'Y':
		return 'у'
	case 'a', 'A':
		return 'а'
	case 'e', 'E':
		return 'е'
	case 'o', 'O':
		return 'о'
	case 'p', 'P':
		return 'р'
	case 'c', 'C':
		return 'с'
	case 'b', 'B':
		return 'в'
	case 'h', 'H':
		return 'н'
	case 'k', 'K':
		return 'к'
	case 'm', 'M':
		return 'м'
	case 't', 'T':
		return 'т'
	default:
		return r
	}
}
