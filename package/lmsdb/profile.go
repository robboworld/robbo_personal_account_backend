package lmsdb

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AuthUserProfile is openedx.auth_user fields used by LK profile and auth.
type AuthUserProfile struct {
	ID          int64
	Username    string
	Email       string
	FirstName   string
	LastName    string
	IsStaff     bool
	IsSuperuser bool
	IsActive    bool
	DateJoined  time.Time
}

func scanAuthUserProfile(row *sql.Row) (*AuthUserProfile, error) {
	var u AuthUserProfile
	var isStaff, isSuper, isActive int
	var dateJoined time.Time
	if err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&isStaff, &isSuper, &isActive, &dateJoined,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.IsStaff = isStaff != 0
	u.IsSuperuser = isSuper != 0
	u.IsActive = isActive != 0
	u.DateJoined = dateJoined
	return &u, nil
}

const authUserProfileSelect = `SELECT id, username, email, first_name, last_name, is_staff, is_superuser, is_active, date_joined
	FROM auth_user`

// LookupAuthUserProfileByID loads profile fields for an LMS user id.
func (r *Reader) LookupAuthUserProfileByID(id int64) (*AuthUserProfile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("lms mysql reader is not configured")
	}
	return scanAuthUserProfile(r.db.QueryRow(authUserProfileSelect+` WHERE id = ? LIMIT 1`, id))
}

// LookupAuthUserProfileByEmail loads profile fields for a user email.
func (r *Reader) LookupAuthUserProfileByEmail(email string) (*AuthUserProfile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("lms mysql reader is not configured")
	}
	return scanAuthUserProfile(r.db.QueryRow(authUserProfileSelect+` WHERE LOWER(email) = LOWER(?) LIMIT 1`, email))
}

// SearchAuthUserProfilesByEmail returns users whose email matches a LIKE pattern.
func (r *Reader) SearchAuthUserProfilesByEmail(emailPattern string, limit, offset int) ([]AuthUserProfile, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("lms mysql reader is not configured")
	}
	if limit < 1 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	pattern := "%" + emailPattern + "%"
	var count int64
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM auth_user WHERE LOWER(email) LIKE LOWER(?)`, pattern,
	).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(
		authUserProfileSelect+` WHERE LOWER(email) LIKE LOWER(?) ORDER BY id ASC LIMIT ? OFFSET ?`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AuthUserProfile
	for rows.Next() {
		var u AuthUserProfile
		var isStaff, isSuper, isActive int
		var dateJoined time.Time
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
			&isStaff, &isSuper, &isActive, &dateJoined,
		); err != nil {
			return nil, 0, err
		}
		u.IsStaff = isStaff != 0
		u.IsSuperuser = isSuper != 0
		u.IsActive = isActive != 0
		u.DateJoined = dateJoined
		out = append(out, u)
	}
	return out, count, rows.Err()
}

func viperWriteDSN() string {
	if v := strings.TrimSpace(os.Getenv("LMS_MYSQL_WRITE_DSN")); v != "" {
		return v
	}
	if v := strings.TrimSpace(viper.GetString("lmsMysql.writeDsn")); v != "" {
		return v
	}
	// Dev fallback: same DSN as reader when write DSN not set.
	if v := strings.TrimSpace(viper.GetString("lmsMysql.dsn")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("LMS_MYSQL_DSN"))
}

// Writer provides write access to openedx.auth_user (profile updates).
type Writer struct {
	db *sql.DB
}

func NewWriterFromConfig() (*Writer, error) {
	dsn := viperWriteDSN()
	if dsn == "" {
		return nil, errors.New("lmsMysql.writeDsn or LMS_MYSQL_WRITE_DSN is not configured")
	}
	db, err := sql.Open("mysql", ensureParseTimeDSN(dsn))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("lms mysql write ping: %w", err)
	}
	return &Writer{db: db}, nil
}

func (w *Writer) Close() error {
	if w == nil || w.db == nil {
		return nil
	}
	return w.db.Close()
}

// UpdateProfile updates email and name fields; username is not changed.
func (w *Writer) UpdateProfile(id int64, email, firstName, lastName string) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	res, err := w.db.Exec(
		`UPDATE auth_user SET email = ?, first_name = ?, last_name = ? WHERE id = ?`,
		email, firstName, lastName, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// MySQL reports 0 rows affected when values are unchanged; not an error.
	}
	return nil
}
