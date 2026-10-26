package oidc

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// DeviceCookieName identifies the browser for the concurrent sign-in limit: every
// browser is one slot, so PCs behind one school NAT no longer share a session.
// It survives sign-out on purpose; signing in again from it reuses its session.
const DeviceCookieName = "lk_device"

// Browsers cap cookie lifetime at 400 days; the cookie is renewed on every sign-in.
const deviceCookieMaxAge = 400 * 24 * 60 * 60

// DeviceID returns the browser's id from the lk_device cookie, issuing a new one when
// it is missing or malformed (the value is client-controlled: only 32 hex chars pass).
func DeviceID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	id, err := c.Cookie(DeviceCookieName)
	if err != nil || !isDeviceID(id) {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return ""
		}
		id = hex.EncodeToString(b)
	}
	SetHTTPOnlyCookie(c, DeviceCookieName, id, deviceCookieMaxAge)
	return id
}

func isDeviceID(v string) bool {
	if len(v) != 32 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
