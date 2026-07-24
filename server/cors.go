package server

import (
	"net"
	"net/url"
	"os"
	"strings"
)

var defaultCORSOrigins = []string{
	"http://0.0.0.0:3030",
	"http://0.0.0.0:3000",
	"http://0.0.0.0:8601",
	"http://localhost:8601",
	"http://127.0.0.1:8601",
	"http://localhost:3030",
	"http://localhost:3000",
	"http://localhost:8080",
	"http://127.0.0.1:3030",
	"http://127.0.0.1:3000",
	"https://scratch.ru",
	"http://scratch.ru",
	"https://scratch-gui.robbo.world",
	"http://scratch-gui.robbo.world",
	"http://127.0.0.1:5001",
	"http://localhost:5001",
	"http://0.0.0.0:5001",
}

func corsOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range corsAllowedOriginsList() {
		if origin == allowed {
			return true
		}
	}
	if os.Getenv("CORS_ALLOW_PRIVATE_NETWORK") == "true" {
		return isPrivateNetworkFrontendOrigin(origin)
	}
	return false
}

func corsAllowedOriginsList() []string {
	origins := append([]string(nil), defaultCORSOrigins...)
	extra := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if extra == "" {
		return origins
	}
	for _, part := range strings.Split(extra, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	return origins
}

func isPrivateNetworkFrontendOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" {
		return false
	}
	switch u.Port() {
	case "3030", "3000", "8601", "5001":
	default:
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "0.0.0.0" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}
