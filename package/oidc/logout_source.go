package oidc

import (
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// LogoutSourceAllowed guards the GET logout routes against cross-site logout (a foreign page
// embedding /auth/oidc/logout as an image or link). The routes stay GET because LMS (redirect
// and IDA_LOGOUT_URI_LIST iframe) and Scratch navigate to them.
//
// A present Origin, or else Referer, must belong to us: a return_to origin (LK, LMS, RS, dev
// ports), the LMS host or its subdomains (MFEs on apps.<lms>), or this backend itself.
// Requests without both headers pass: Django's default Referrer-Policy (same-origin) strips
// Referer from the LMS logout redirect and iframe, so their absence is normal.
func LogoutSourceAllowed(origin, referer, requestOrigin string) bool {
	source := strings.TrimSpace(origin)
	if source == "" {
		ref := strings.TrimSpace(referer)
		if ref == "" {
			return true
		}
		parsed, err := url.Parse(ref)
		if err != nil || parsed.Host == "" {
			return false
		}
		source = parsed.Scheme + "://" + parsed.Host
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false // includes Origin: null (sandboxed iframes, file://)
	}
	source = strings.ToLower(parsed.Scheme + "://" + parsed.Host)
	if strings.EqualFold(source, requestOrigin) || originAllowed(source) {
		return true
	}
	return underLMSHost(parsed.Hostname())
}

func underLMSHost(host string) bool {
	host = strings.ToLower(host)
	for _, raw := range []string{os.Getenv("LMS_URL"), os.Getenv("LMS_PUBLIC_URL"), viper.GetString("lms.url")} {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		lms := strings.ToLower(parsed.Hostname())
		if host == lms || strings.HasSuffix(host, "."+lms) {
			return true
		}
	}
	return false
}
