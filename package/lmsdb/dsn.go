package lmsdb

import "strings"

func ensureParseTimeDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return dsn
	}
	if !strings.Contains(dsn, "parseTime=") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}
	// Avoid mojibake for Cyrillic names (auth_userprofile.name, etc.).
	if !strings.Contains(dsn, "charset=") {
		dsn += "&charset=utf8mb4"
	}
	if !strings.Contains(dsn, "collation=") {
		dsn += "&collation=utf8mb4_unicode_ci"
	}
	if !strings.Contains(dsn, "timeout=") {
		dsn += "&timeout=5s"
	}
	if !strings.Contains(dsn, "readTimeout=") {
		dsn += "&readTimeout=5s"
	}
	if !strings.Contains(dsn, "writeTimeout=") {
		dsn += "&writeTimeout=5s"
	}
	return dsn
}
