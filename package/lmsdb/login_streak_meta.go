package lmsdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const metaKeyLoginStreak = "lk_login_streak"

// LoginStreakMeta is stored under auth_userprofile.meta["lk_login_streak"].
type LoginStreakMeta struct {
	Current       int    `json:"current"`
	Longest       int    `json:"longest"`
	LastLoginDate string `json:"last_login_date"` // YYYY-MM-DD in user timezone
	Timezone      string `json:"timezone"`
}

// ParseLoginStreakFromMeta reads lk_login_streak from meta JSON.
func ParseLoginStreakFromMeta(meta string) LoginStreakMeta {
	meta = strings.TrimSpace(meta)
	if meta == "" || meta == "{}" {
		return LoginStreakMeta{}
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(meta), &payload); err != nil {
		return LoginStreakMeta{}
	}
	raw, ok := payload[metaKeyLoginStreak]
	if !ok || len(raw) == 0 {
		return LoginStreakMeta{}
	}
	var streak LoginStreakMeta
	if err := json.Unmarshal(raw, &streak); err != nil {
		return LoginStreakMeta{}
	}
	if streak.Current < 0 {
		streak.Current = 0
	}
	if streak.Longest < 0 {
		streak.Longest = 0
	}
	return streak
}

// SetLoginStreakInMeta merges lk_login_streak into meta JSON. Preserves other keys.
func SetLoginStreakInMeta(meta string, streak LoginStreakMeta) (string, error) {
	payload := map[string]interface{}{}
	meta = strings.TrimSpace(meta)
	if meta != "" && meta != "{}" {
		if err := json.Unmarshal([]byte(meta), &payload); err != nil {
			payload = map[string]interface{}{}
		}
	}
	if streak.Current < 0 {
		streak.Current = 0
	}
	if streak.Longest < 0 {
		streak.Longest = 0
	}
	if streak.Longest < streak.Current {
		streak.Longest = streak.Current
	}
	payload[metaKeyLoginStreak] = streak
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetLoginStreak loads streak from auth_userprofile.meta for an LMS user.
func (r *Reader) GetLoginStreak(userID int64) (LoginStreakMeta, error) {
	if r == nil || r.db == nil {
		return LoginStreakMeta{}, errors.New("lms mysql reader is not configured")
	}
	if userID <= 0 {
		return LoginStreakMeta{}, nil
	}
	var meta sql.NullString
	err := r.db.QueryRow(
		`SELECT COALESCE(p.meta, '') FROM auth_userprofile p WHERE p.user_id = ? LIMIT 1`,
		userID,
	).Scan(&meta)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginStreakMeta{}, nil
	}
	if err != nil {
		return LoginStreakMeta{}, err
	}
	if !meta.Valid {
		return LoginStreakMeta{}, nil
	}
	return ParseLoginStreakFromMeta(meta.String), nil
}

// UpsertLoginStreak writes streak into auth_userprofile.meta (merge).
func (w *Writer) UpsertLoginStreak(userID int64, streak LoginStreakMeta) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	if userID <= 0 {
		return sql.ErrNoRows
	}

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var meta sql.NullString
	err = tx.QueryRow(`SELECT meta FROM auth_userprofile WHERE user_id = ? LIMIT 1`, userID).Scan(&meta)
	if errors.Is(err, sql.ErrNoRows) {
		newMeta, mergeErr := SetLoginStreakInMeta("{}", streak)
		if mergeErr != nil {
			return mergeErr
		}
		_, err = tx.Exec(
			`INSERT INTO auth_userprofile (name, meta, courseware, language, location, user_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			"", newMeta, defaultProfileCourseware, "", defaultProfileLocation, userID,
		)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	if err != nil {
		return err
	}

	current := ""
	if meta.Valid {
		current = meta.String
	}
	newMeta, err := SetLoginStreakInMeta(current, streak)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE auth_userprofile SET meta = ? WHERE user_id = ?`, newMeta, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ParseLMSUserID converts string LMS id to int64.
func ParseLMSUserID(lmsUserID string) (int64, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(lmsUserID, 10, 64)
}

// FormatLoginDate formats a calendar date as YYYY-MM-DD.
func FormatLoginDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// ParseLoginDate parses YYYY-MM-DD as UTC midnight.
func ParseLoginDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
