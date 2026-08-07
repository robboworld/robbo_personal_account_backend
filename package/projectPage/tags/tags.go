package tags

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

const (
	MaxTags   = 5
	MinTagLen = 1
	MaxTagLen = 25
)

var tagPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$|^[a-z0-9]$`)

// NormalizeOne converts a single tag to a GitHub-like slug.
// Returns empty string if the input cannot be normalized to a valid tag.
func NormalizeOne(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	if utf8.RuneCountInString(out) > MaxTagLen {
		runes := []rune(out)
		out = string(runes[:MaxTagLen])
		out = strings.Trim(out, "-")
	}
	if !tagPattern.MatchString(out) {
		return ""
	}
	n := utf8.RuneCountInString(out)
	if n < MinTagLen || n > MaxTagLen {
		return ""
	}
	return out
}

// NormalizeList normalizes, dedupes and caps tag list. Empty input → empty slice.
// Invalid non-empty raw tags that cannot be normalized cause ErrBadRequest.
func NormalizeList(raw []string) ([]string, error) {
	if raw == nil {
		return []string{}, nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		n := NormalizeOne(trimmed)
		if n == "" {
			return nil, fmt.Errorf("%w: invalid tag %q", projectPage.ErrBadRequest, trimmed)
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
		if len(out) > MaxTags {
			return nil, fmt.Errorf("%w: at most %d tags allowed", projectPage.ErrBadRequest, MaxTags)
		}
	}
	return out, nil
}

// NormalizeFilterList normalizes a catalog tag filter (AND of exact tags).
// Empty entries are skipped; invalid tags return ErrBadRequest.
func NormalizeFilterList(raw []string) ([]string, error) {
	return NormalizeList(raw)
}
