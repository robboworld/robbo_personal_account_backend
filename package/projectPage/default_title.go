package projectPage

import (
	"strings"
)

const DefaultLanguage = "ru"

// DefaultProjectTitle returns the localized default project name for the UI language.
func DefaultProjectTitle(lang string) string {
	switch NormalizeLanguage(lang) {
	case "zh":
		return "未命名"
	case "en":
		return "Untitled"
	default:
		return "Безымянный"
	}
}

// NormalizeLanguage maps Accept-Language / UI codes to ru|en|zh.
func NormalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return DefaultLanguage
	}
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		lang = lang[:i]
	}
	switch lang {
	case "ru", "en", "zh":
		return lang
	default:
		return DefaultLanguage
	}
}

// LocaleFromAcceptLanguage picks the first supported language from an Accept-Language header.
func LocaleFromAcceptLanguage(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return DefaultLanguage
	}
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(part)
		if i := strings.Index(tag, ";"); i >= 0 {
			tag = strings.TrimSpace(tag[:i])
		}
		if tag == "" || tag == "*" {
			continue
		}
		normalized := NormalizeLanguage(tag)
		// NormalizeLanguage maps unknown → DefaultLanguage; only accept if the primary
		// subtag itself is supported, otherwise keep scanning.
		primary := strings.ToLower(tag)
		if i := strings.IndexAny(primary, "-_"); i > 0 {
			primary = primary[:i]
		}
		switch primary {
		case "ru", "en", "zh":
			return normalized
		}
	}
	return DefaultLanguage
}
