package lmsdb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

// Reader provides read-only SQL access to Open edX MySQL (auth_user, enrollments).
// Use only from workers/scripts — not from HTTP GraphQL handlers.
type Reader struct {
	db *sql.DB
}

func NewReaderFromConfig() (*Reader, error) {
	dsn := viper.GetString("lmsMysql.dsn")
	if dsn == "" {
		dsn = viper.GetString("LMS_MYSQL_DSN")
	}
	if dsn == "" {
		return nil, errors.New("lmsMysql.dsn or LMS_MYSQL_DSN is not configured")
	}
	db, err := sql.Open("mysql", ensureParseTimeDSN(dsn))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("lms mysql ping: %w", err)
	}
	return &Reader{db: db}, nil
}

func (r *Reader) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

type AuthUserRow struct {
	ID       int64
	Username string
	Email    string
}

// AuthUserLogin row used for email/password authentication against LMS dump.
type AuthUserLogin struct {
	ID          int64
	Username    string
	Email       string
	Password    string
	IsStaff     bool
	IsSuperuser bool
	IsActive    bool
}

// LookupAuthUserForLogin loads auth_user by email for password verification.
func (r *Reader) LookupAuthUserForLogin(email string) (*AuthUserLogin, error) {
	const q = `SELECT id, username, email, password, is_staff, is_superuser, is_active
		FROM auth_user WHERE LOWER(email) = LOWER(?) LIMIT 1`
	row := r.db.QueryRow(q, email)
	var u AuthUserLogin
	var isStaff, isSuper, isActive int
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &isStaff, &isSuper, &isActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.IsStaff = isStaff != 0
	u.IsSuperuser = isSuper != 0
	u.IsActive = isActive != 0
	return &u, nil
}

// Authenticate verifies email/password against openedx.auth_user (Django pbkdf2_sha256).
func (r *Reader) Authenticate(email, password string) (*AuthUserLogin, error) {
	u, err := r.LookupAuthUserForLogin(email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, auth.ErrUserNotFound
	}
	if !u.IsActive {
		return nil, auth.ErrUserInactive
	}
	if !VerifyDjangoPassword(password, u.Password) {
		return nil, auth.ErrInvalidCredentials
	}
	return u, nil
}

// LookupAuthUserByID returns auth_user username/email for an LMS user id.
func (r *Reader) LookupAuthUserByID(id int64) (*AuthUserRow, error) {
	const q = `SELECT id, username, email FROM auth_user WHERE id = ? LIMIT 1`
	row := r.db.QueryRow(q, id)
	var u AuthUserRow
	if err := row.Scan(&u.ID, &u.Username, &u.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// LookupAuthUserByEmail returns auth_user.id for an active user email.
func (r *Reader) LookupAuthUserByEmail(email string) (*AuthUserRow, error) {
	const q = `SELECT id, username, email FROM auth_user WHERE email = ? AND is_active = 1 LIMIT 1`
	row := r.db.QueryRow(q, email)
	var u AuthUserRow
	if err := row.Scan(&u.ID, &u.Username, &u.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// HasEnrollment checks student_courseenrollment for idempotency before API enroll.
func (r *Reader) HasEnrollment(userID int64, courseID string) (bool, error) {
	const q = `SELECT 1 FROM student_courseenrollment WHERE user_id = ? AND course_id = ? AND is_active = 1 LIMIT 1`
	row := r.db.QueryRow(q, userID, courseID)
	var one int
	err := row.Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
