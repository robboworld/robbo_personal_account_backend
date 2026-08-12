package lmsdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/avatars"
)

const metaKeyAvatarID = "lk_avatar_id"

// ParseAvatarIDFromMeta reads lk_avatar_id from auth_userprofile.meta JSON.
// Invalid or unknown values are treated as unset.
func ParseAvatarIDFromMeta(meta string) string {
	meta = strings.TrimSpace(meta)
	if meta == "" || meta == "{}" {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(meta), &payload); err != nil {
		return ""
	}
	raw, ok := payload[metaKeyAvatarID]
	if !ok || raw == nil {
		return ""
	}
	id, ok := raw.(string)
	if !ok {
		return ""
	}
	id = strings.TrimSpace(id)
	if !avatars.IsValidAvatarID(id) {
		return ""
	}
	return id
}

// SetAvatarIDInMeta merges lk_avatar_id into meta JSON.
// Empty avatarID removes the key. Preserves other meta fields.
func SetAvatarIDInMeta(meta, avatarID string) (string, error) {
	payload := map[string]interface{}{}
	meta = strings.TrimSpace(meta)
	if meta != "" && meta != "{}" {
		if err := json.Unmarshal([]byte(meta), &payload); err != nil {
			// Corrupt meta: start fresh rather than failing the avatar write.
			payload = map[string]interface{}{}
		}
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		delete(payload, metaKeyAvatarID)
	} else {
		if !avatars.IsValidAvatarID(avatarID) {
			return "", avatars.ErrInvalidAvatarID
		}
		payload[metaKeyAvatarID] = avatarID
	}

	if len(payload) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UpdateProfileAvatarID sets or clears lk_avatar_id in auth_userprofile.meta.
func (w *Writer) UpdateProfileAvatarID(userID int64, avatarID string) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	if userID <= 0 {
		return sql.ErrNoRows
	}

	normalized, err := avatars.NormalizeAvatarID(avatarID)
	if err != nil {
		return err
	}

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var meta sql.NullString
	err = tx.QueryRow(`SELECT meta FROM auth_userprofile WHERE user_id = ? LIMIT 1`, userID).Scan(&meta)
	if errors.Is(err, sql.ErrNoRows) {
		newMeta, mergeErr := SetAvatarIDInMeta("{}", normalized)
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
	newMeta, err := SetAvatarIDInMeta(current, normalized)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE auth_userprofile SET meta = ? WHERE user_id = ?`, newMeta, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
