// Backfill auth_userprofile.name from auth_user first_name/last_name when profile name is empty.
//
//	go run ./scripts/backfill_auth_userprofile_name/main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := flag.String("dsn", envOr("LMS_MYSQL_ROOT_DSN", "root:lms_root_change_me@tcp(127.0.0.1:3307)/openedx"), "MySQL DSN with write access")
	dryRun := flag.Bool("dry-run", false, "print actions without writing")
	flag.Parse()

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("ping: ", err)
	}

	rows, err := db.Query(`
		SELECT u.id, u.first_name, u.last_name, COALESCE(p.id, 0), COALESCE(p.name, '')
		FROM auth_user u
		LEFT JOIN auth_userprofile p ON p.user_id = u.id
		ORDER BY u.id`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var updated, inserted int
	for rows.Next() {
		var userID int64
		var firstName, lastName, profileName string
		var profileID int64
		if err := rows.Scan(&userID, &firstName, &lastName, &profileID, &profileName); err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(profileName) != "" {
			continue
		}
		fullName := strings.TrimSpace(strings.Join([]string{strings.TrimSpace(firstName), strings.TrimSpace(lastName)}, " "))
		if fullName == "" {
			continue
		}
		if *dryRun {
			fmt.Printf("user_id=%d fullName=%q profile_exists=%v\n", userID, fullName, profileID > 0)
			continue
		}
		if profileID > 0 {
			if _, err := db.Exec(`UPDATE auth_userprofile SET name = ? WHERE user_id = ?`, fullName, userID); err != nil {
				log.Fatal(err)
			}
			updated++
		} else {
			if _, err := db.Exec(
				`INSERT INTO auth_userprofile (name, meta, courseware, language, location, user_id) VALUES (?, '', 'course.xml', '', '', ?)`,
				fullName, userID,
			); err != nil {
				log.Fatal(err)
			}
			inserted++
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("done: updated=%d inserted=%d dry_run=%v\n", updated, inserted, *dryRun)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
