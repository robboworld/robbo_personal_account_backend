package lmsdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/spf13/viper"
)

// AuthUserProfile is openedx.auth_user + auth_userprofile fields used by LK profile and auth.
type AuthUserProfile struct {
	ID               int64
	Username         string
	Email            string
	FirstName        string
	LastName         string
	ProfileName      string
	LevelOfEducation sql.NullString
	Country          sql.NullString
	YearOfBirth      sql.NullInt64
	Gender           sql.NullString
	Language         sql.NullString
	Meta             string
	IsStaff          bool
	IsSuperuser      bool
	IsActive         bool
	DateJoined       time.Time
}

// ProfileExtendedUpdate holds optional auth_userprofile demographic fields.
type ProfileExtendedUpdate struct {
	LevelOfEducation string
	Country          string
	YearOfBirth      *int
	Gender           string
	Language         string
}

const authUserProfileJoinSelect = `SELECT u.id, u.username, u.email, u.first_name, u.last_name,
	u.is_staff, u.is_superuser, u.is_active, u.date_joined,
	COALESCE(p.name, '') AS profile_name,
	p.level_of_education, p.country, p.year_of_birth, p.gender, p.language,
	COALESCE(p.meta, '') AS meta
FROM auth_user u
LEFT JOIN auth_userprofile p ON p.user_id = u.id`

func scanAuthUserProfile(row *sql.Row) (*AuthUserProfile, error) {
	var u AuthUserProfile
	var isStaff, isSuper, isActive int
	var dateJoined time.Time
	if err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&isStaff, &isSuper, &isActive, &dateJoined, &u.ProfileName,
		&u.LevelOfEducation, &u.Country, &u.YearOfBirth, &u.Gender, &u.Language, &u.Meta,
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

func scanAuthUserProfileRows(rows *sql.Rows) (AuthUserProfile, error) {
	var u AuthUserProfile
	var isStaff, isSuper, isActive int
	var dateJoined time.Time
	if err := rows.Scan(
		&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&isStaff, &isSuper, &isActive, &dateJoined, &u.ProfileName,
		&u.LevelOfEducation, &u.Country, &u.YearOfBirth, &u.Gender, &u.Language, &u.Meta,
	); err != nil {
		return AuthUserProfile{}, err
	}
	u.IsStaff = isStaff != 0
	u.IsSuperuser = isSuper != 0
	u.IsActive = isActive != 0
	u.DateJoined = dateJoined
	return u, nil
}

// LookupAuthUserProfileByID loads profile fields for an LMS user id.
func (r *Reader) LookupAuthUserProfileByID(id int64) (*AuthUserProfile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("lms mysql reader is not configured")
	}
	return scanAuthUserProfile(r.db.QueryRow(authUserProfileJoinSelect+` WHERE u.id = ? LIMIT 1`, id))
}

// LookupAuthUserProfileByEmail loads profile fields for a user email.
func (r *Reader) LookupAuthUserProfileByEmail(email string) (*AuthUserProfile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("lms mysql reader is not configured")
	}
	return scanAuthUserProfile(r.db.QueryRow(authUserProfileJoinSelect+` WHERE LOWER(u.email) = LOWER(?) LIMIT 1`, email))
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
		authUserProfileJoinSelect+` WHERE LOWER(u.email) LIKE LOWER(?) ORDER BY u.id ASC LIMIT ? OFFSET ?`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AuthUserProfile
	for rows.Next() {
		u, err := scanAuthUserProfileRows(rows)
		if err != nil {
			return nil, 0, err
		}
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
	if v := strings.TrimSpace(viper.GetString("lmsMysql.dsn")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("LMS_MYSQL_DSN"))
}

// Writer provides write access to openedx.auth_user and auth_userprofile.
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

// DB exposes the underlying sql.DB for maintenance scripts.
func (w *Writer) DB() *sql.DB {
	if w == nil {
		return nil
	}
	return w.db
}

// TouchLastLogin sets auth_user.last_login to the current UTC time.
func (w *Writer) TouchLastLogin(userID int64) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	_, err := w.db.Exec(
		`UPDATE auth_user SET last_login = ? WHERE id = ?`,
		time.Now().UTC(), userID,
	)
	return err
}

// UpdateAccountEmail updates auth_user.email only.
func (w *Writer) UpdateAccountEmail(id int64, email string) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	res, err := w.db.Exec(`UPDATE auth_user SET email = ? WHERE id = ?`, email, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

const defaultProfileCourseware = "course.xml"
const defaultProfileLocation = ""

func buildCompanyMeta(company string) (string, error) {
	company = strings.TrimSpace(company)
	if company == "" {
		return "", auth.ErrCompanyRequired
	}
	b, err := json.Marshal(map[string]string{"company": company})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func nullString(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

// profileLanguage returns a value for auth_userprofile.language (NOT NULL in openedx).
func profileLanguage(s string) string {
	return strings.TrimSpace(s)
}

func nullInt(year *int) interface{} {
	if year == nil || *year <= 0 {
		return nil
	}
	return *year
}

// UpsertProfileName writes full display name to auth_userprofile.name.
func (w *Writer) UpsertProfileName(userID int64, fullName string) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	res, err := w.db.Exec(
		`UPDATE auth_userprofile SET name = ? WHERE user_id = ?`,
		fullName, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = w.db.Exec(
		`INSERT INTO auth_userprofile (name, meta, courseware, language, location, user_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fullName, "", defaultProfileCourseware, "", defaultProfileLocation, userID,
	)
	return err
}

// UpdateProfile updates auth_user email and auth_userprofile demographic fields.
func (w *Writer) UpdateProfile(id int64, email, fullName string, ext ProfileExtendedUpdate) error {
	if w == nil || w.db == nil {
		return errors.New("lms mysql writer is not configured")
	}
	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`UPDATE auth_user SET email = ? WHERE id = ?`, email, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var one int
		if err := tx.QueryRow(`SELECT 1 FROM auth_user WHERE id = ? LIMIT 1`, id).Scan(&one); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return err
		}
	}

	res, err = tx.Exec(
		`UPDATE auth_userprofile SET name = ?, level_of_education = ?, country = ?,
		 year_of_birth = ?, gender = ?, language = ?
		 WHERE user_id = ?`,
		fullName,
		nullString(ext.LevelOfEducation),
		nullString(ext.Country),
		nullInt(ext.YearOfBirth),
		nullString(ext.Gender),
		profileLanguage(ext.Language),
		id,
	)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var profileExists int
		err = tx.QueryRow(`SELECT 1 FROM auth_userprofile WHERE user_id = ? LIMIT 1`, id).Scan(&profileExists)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.Exec(
				`INSERT INTO auth_userprofile (name, meta, courseware, language, location,
				 level_of_education, country, year_of_birth, gender, user_id)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				fullName, "", defaultProfileCourseware, profileLanguage(ext.Language), defaultProfileLocation,
				nullString(ext.LevelOfEducation),
				nullString(ext.Country),
				nullInt(ext.YearOfBirth),
				nullString(ext.Gender),
				id,
			)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateUserWithProfile inserts auth_user and auth_userprofile in one transaction.
func (w *Writer) CreateUserWithProfile(username, email, passwordHash, fullName, company string) (int64, error) {
	if w == nil || w.db == nil {
		return 0, errors.New("lms mysql writer is not configured")
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)
	if username == "" || email == "" || passwordHash == "" {
		return 0, auth.ErrUserNotFound
	}

	meta, err := buildCompanyMeta(company)
	if err != nil {
		return 0, err
	}

	tx, err := w.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var emailCount int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM auth_user WHERE LOWER(email) = LOWER(?)`,
		email,
	).Scan(&emailCount); err != nil {
		return 0, err
	}
	if emailCount > 0 {
		return 0, auth.ErrEmailAlreadyExist
	}

	var usernameCount int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM auth_user WHERE username = ?`,
		username,
	).Scan(&usernameCount); err != nil {
		return 0, err
	}
	if usernameCount > 0 {
		return 0, auth.ErrUsernameAlreadyExist
	}

	now := time.Now().UTC()
	res, err := tx.Exec(
		`INSERT INTO auth_user (password, last_login, is_superuser, username, first_name, last_name, email, is_staff, is_active, date_joined)
		 VALUES (?, NULL, 0, ?, '', '', ?, 0, 1, ?)`,
		passwordHash, username, email, now,
	)
	if err != nil {
		return 0, err
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(
		`INSERT INTO auth_userprofile (name, meta, courseware, language, location, user_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fullName, meta, defaultProfileCourseware, "", defaultProfileLocation, userID,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return userID, nil
}
