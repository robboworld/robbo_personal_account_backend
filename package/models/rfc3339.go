package models

import "time"

// RFC3339UTC is a JavaScript-parseable timestamp. Go's Time.String()
// ("2006-01-02 15:04:05.937423 +0300 MSK") is not.
func RFC3339UTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
