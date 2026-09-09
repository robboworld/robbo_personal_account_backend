package oidc

import (
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const defaultSafeReturnTo = "/home"

// SanitizeReturnTo accepts a relative path or an absolute URL whose origin is
// on the allowlist (LK frontend, LMS, Scratch editor, extra env origins).
func SanitizeReturnTo(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSafeReturnTo
	}
	if strings.ContainsAny(raw, "\r\n") {
		return defaultSafeReturnTo
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
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
	add(os.Getenv("OIDC_FRONTEND_BASE_URL"))
	add("http://localhost:3030")
	add("http://127.0.0.1:3030")
	add("http://localhost:8601")
	add("http://127.0.0.1:8601")
	add("http://localhost:5001")
	add("http://127.0.0.1:5001")
	add("https://scratch.example.com")
	add("http://scratch.example.com")

	for _, part := range strings.Split(os.Getenv("OIDC_RETURN_TO_ORIGINS"), ",") {
		add(part)
	}
	for _, part := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		add(part)
	}
	return out
}
