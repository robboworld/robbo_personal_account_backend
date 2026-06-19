// Seed a dev Open edX auth_user for local LK password login.
// Usage:
//
//	go run ./scripts/seed_lms_dev_user/main.go
//	go run ./scripts/seed_lms_dev_user/main.go -dsn 'root:lms_root_change_me@tcp(127.0.0.1:3307)/openedx'
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
)

func main() {
	username := flag.String("username", "1@1", "auth_user.username")
	email := flag.String("email", "1@1", "auth_user.email (login field in LK)")
	password := flag.String("password", "123", "plain password")
	fullName := flag.String("name", "Ivan 1234", "auth_userprofile.name (full display name)")
	flag.Parse()

	encoded, err := lmsdb.EncodeDjangoPassword(*password)
	if err != nil {
		log.Fatal(err)
	}

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close()

	userID, err := upsertDevUser(writer, encoded, *username, *email, strings.TrimSpace(*fullName))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("edx_user_id=%d fullName=%q\n", userID, *fullName)
	fmt.Println("LK login: email", *email, "password", *password, "role: ученик (student)")
}

func upsertDevUser(writer *lmsdb.Writer, encoded, username, email, fullName string) (int64, error) {
	db := writer.DB()
	var existing int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM auth_user WHERE LOWER(email) = LOWER(?) OR username = ?`,
		email, username,
	).Scan(&existing); err != nil {
		return 0, err
	}

	var userID int64
	if existing > 0 {
		if err := db.QueryRow(
			`SELECT id FROM auth_user WHERE LOWER(email) = LOWER(?) LIMIT 1`, email,
		).Scan(&userID); err != nil {
			return 0, err
		}
		if _, err := db.Exec(
			`UPDATE auth_user SET password = ?, is_active = 1, is_staff = 0, is_superuser = 0 WHERE id = ?`,
			encoded, userID,
		); err != nil {
			return 0, err
		}
		fmt.Printf("updated user email=%s username=%s\n", email, username)
	} else {
		var err error
		userID, err = writer.CreateUserWithProfile(username, email, encoded, fullName, "", false)
		if err != nil {
			return 0, err
		}
		fmt.Printf("created user email=%s username=%s\n", email, username)
		return userID, nil
	}

	if err := writer.UpsertProfileName(userID, fullName); err != nil {
		return 0, err
	}
	return userID, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
