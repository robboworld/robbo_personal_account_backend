// Seed a dev Open edX auth_user for local LK password login.
// Usage:
//
//	go run ./scripts/seed_lms_dev_user/main.go
//	go run ./scripts/seed_lms_dev_user/main.go -dsn 'root:lms_root_change_me@tcp(127.0.0.1:3307)/openedx'
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := flag.String("dsn", envOr("LMS_MYSQL_ROOT_DSN", "root:lms_root_change_me@tcp(127.0.0.1:3307)/openedx"), "MySQL DSN with INSERT rights")
	username := flag.String("username", "1@1", "auth_user.username")
	email := flag.String("email", "1@1", "auth_user.email (login field in LK)")
	password := flag.String("password", "123", "plain password")
	flag.Parse()

	encoded, err := lmsdb.EncodeDjangoPassword(*password)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("mysql ping: ", err)
	}

	var existing int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM auth_user WHERE LOWER(email) = LOWER(?) OR username = ?`,
		*email, *username,
	).Scan(&existing); err != nil {
		log.Fatal(err)
	}
	if existing > 0 {
		_, err = db.Exec(
			`UPDATE auth_user SET password = ?, is_active = 1, is_staff = 0, is_superuser = 0 WHERE LOWER(email) = LOWER(?) OR username = ?`,
			encoded, *email, *username,
		)
		if err != nil {
			log.Fatal("update: ", err)
		}
		fmt.Printf("updated user email=%s username=%s password=%q\n", *email, *username, *password)
	} else {
		_, err = db.Exec(
			`INSERT INTO auth_user (password, last_login, is_superuser, username, first_name, last_name, email, is_staff, is_active, date_joined)
			 VALUES (?, NULL, 0, ?, '', '', ?, 0, 1, ?)`,
			encoded, *username, *email, time.Now().UTC(),
		)
		if err != nil {
			log.Fatal("insert: ", err)
		}
		fmt.Printf("created user email=%s username=%s password=%q\n", *email, *username, *password)
	}

	var id int64
	if err := db.QueryRow(`SELECT id FROM auth_user WHERE LOWER(email) = LOWER(?) LIMIT 1`, *email).Scan(&id); err != nil {
		log.Fatal("lookup id: ", err)
	}
	fmt.Printf("edx_user_id=%d\n", id)
	fmt.Println("LK login: email", *email, "password", *password, "role: ученик (student)")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
