package oidc

import (
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const defaultSafeReturnTo = "/home"

// Aliases avoid nested query strings in LMS redirect_url (& → &amp; breaks return_to).
func resolveReturnToAlias(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lms", "lms_landing", "openedx":
		lms := strings.TrimSpace(os.Getenv("LMS_URL"))
		if lms == "" {
			lms = strings.TrimSpace(viper.GetString("lms.url"))
		}
		if lms == "" {
			lms = strings.TrimSpace(viper.GetString("lms.publicUrl"))
		}
		if lms == "" {
			return defaultSafeReturnTo
		}
		return strings.TrimRight(lms, "/") + "/"
	default:
		return raw
	}
}

// SanitizeReturnTo accepts a relative path or an absolute URL whose origin is
// on the allowlist (LK frontend, LMS, Scratch editor, extra env origins).
func SanitizeReturnTo(raw string) string {
	raw = resolveReturnToAlias(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSafeReturnTo
	}
	if unsafeReturnTo(raw) {
		return defaultSafeReturnTo
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		// Must stay a same-origin path: no scheme or host after parsing.
		if parsed, err := url.Parse(raw); err != nil || parsed.Scheme != "" || parsed.Host != "" {
			return defaultSafeReturnTo
		}
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return defaultSafeReturnTo
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return defaultSafeReturnTo
	}
	if originAllowed(parsed.Scheme + "://" + parsed.Host) {
		return raw
	}
	return defaultSafeReturnTo
}

// unsafeReturnTo rejects control characters and backslashes (raw or percent-encoded):
// browsers treat "/\\evil.com" like "//evil.com", turning a relative path into another origin.
func unsafeReturnTo(raw string) bool {
	for _, r := range raw {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return true
		}
	}
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "%5c") || strings.Contains(lower, "%09") ||
		strings.Contains(lower, "%0a") || strings.Contains(lower, "%0d")
}

func originAllowed(origin string) bool {
	origin = strings.TrimRight(strings.ToLower(origin), "/")
	for _, allowed := range returnToOrigins() {
		if origin == allowed {
			return true
		}
	}
	return false
}

func returnToOrigins() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return
		}
		if parsed.Scheme == "" {
			parsed.Scheme = "http"
		}
		origin := strings.TrimRight(strings.ToLower(parsed.Scheme+"://"+parsed.Host), "/")
		if _, ok := seen[origin]; ok {
			return
		}
		seen[origin] = struct{}{}
		out = append(out, origin)
	}

	add(viper.GetString("oidc.frontendBaseUrl"))
	add(viper.GetString("oidc.postLogoutRedirectUri"))
	add(os.Getenv("LMS_URL"))
	add(os.Getenv("LMS_PUBLIC_URL"))
	add(os.Getenv("ROBBO_RS_URL"))
	add(os.Getenv("OIDC_FRONTEND_BASE_URL"))
	add("http://localhost:3030")
	add("http://127.0.0.1:3030")
	add("http://localhost:8601")
	add("http://127.0.0.1:8601")
	add("http://localhost:5001")
	add("http://127.0.0.1:5001")

	for _, part := range strings.Split(os.Getenv("OIDC_RETURN_TO_ORIGINS"), ",") {
		add(part)
	}
	for _, part := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		add(part)
	}
	return out
}

// ProductLanding is the post-logout URL for a product (no query string — safe as LMS redirect_url).
func ProductLanding(product string) string {
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "lms", "openedx":
		return resolveReturnToAlias("lms")
	case "rs", "scratch":
		rs := strings.TrimSpace(os.Getenv("ROBBO_RS_URL"))
		if rs == "" {
			rs = "http://localhost:8601"
		}
		return strings.TrimRight(rs, "/") + "/"
	default:
		frontend := strings.TrimSpace(viper.GetString("oidc.frontendBaseUrl"))
		if frontend == "" {
			frontend = "http://localhost:3030"
		}
		return strings.TrimRight(frontend, "/") + "/"
	}
}
