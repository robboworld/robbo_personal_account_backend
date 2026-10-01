package server

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// trustedProxies lists reverse-proxy IPs/CIDRs whose X-Forwarded-For / X-Real-IP is honoured
// (server.trustedProxies in config or TRUSTED_PROXIES, comma-separated).
// Empty means no proxy is trusted: the client IP is the TCP peer address.
func trustedProxies() []string {
	var out []string
	add := func(raw string) {
		for _, part := range strings.Split(raw, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	for _, v := range viper.GetStringSlice("server.trustedProxies") {
		add(v)
	}
	add(os.Getenv("TRUSTED_PROXIES"))
	return out
}

// applyTrustedProxies makes c.ClientIP() trust forwarding headers only from configured proxies
// (gin trusts every peer by default, which lets clients spoof their IP).
func applyTrustedProxies(engine *gin.Engine) error {
	return engine.SetTrustedProxies(trustedProxies())
}
