// Package dbmigrate applies the backend's own schema changes to the projects and licensing
// databases at startup. The SQL in robbo_*_db/init only runs on an empty volume, so before
// this every existing database had to be patched by hand and drifted from the code.
//
// Rules: files are migrations/<db>/NNNN_name.sql, applied once in name order and recorded in
// schema_migrations; each file must be idempotent (a fresh volume also ran robbo_*_db/init).
// DB_AUTO_MIGRATE=false turns it off.
package dbmigrate

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

//go:embed migrations
var files embed.FS

// Databases and the config key of their DSN.
var databases = []struct{ name, dsnKey string }{
	{"licensing", "licensingPostgres.postgresDsn"},
	{"projects", "projectsPostgres.postgresDsn"},
}

// ApplyConfigured migrates every configured database; a database without a DSN is skipped.
func ApplyConfigured() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DB_AUTO_MIGRATE")), "false") {
		log.Println("dbmigrate: disabled (DB_AUTO_MIGRATE=false)")
		return nil
	}
	for _, d := range databases {
		dsn := strings.TrimSpace(viper.GetString(d.dsnKey))
		if dsn == "" {
			continue
		}
		db, err := db_client.OpenByDSN(dsn)
		if err != nil {
			return fmt.Errorf("dbmigrate %s: open: %w", d.name, err)
		}
		if err := Apply(db, d.name); err != nil {
			return err
		}
	}
	return nil
}

// Apply runs the not yet applied migrations of one database.
func Apply(db *gorm.DB, name string) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`).Error; err != nil {
		return fmt.Errorf("dbmigrate %s: schema_migrations: %w", name, err)
	}
	var applied []string
	if err := db.Raw(`SELECT version FROM schema_migrations`).Scan(&applied).Error; err != nil {
		return fmt.Errorf("dbmigrate %s: read applied: %w", name, err)
	}
	all, err := versions(name)
	if err != nil {
		return err
	}
	for _, version := range pending(all, applied) {
		body, err := files.ReadFile(path.Join("migrations", name, version))
		if err != nil {
			return fmt.Errorf("dbmigrate %s: read %s: %w", name, version, err)
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(body)).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version).Error
		})
		if err != nil {
			return fmt.Errorf("dbmigrate %s: %s: %w", name, version, err)
		}
		log.Printf("dbmigrate %s: applied %s", name, version)
	}
	return nil
}

// versions lists the .sql files of a database in apply order.
func versions(name string) ([]string, error) {
	entries, err := fs.ReadDir(files, path.Join("migrations", name))
	if err != nil {
		return nil, fmt.Errorf("dbmigrate %s: list: %w", name, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func pending(all, applied []string) []string {
	done := make(map[string]bool, len(applied))
	for _, v := range applied {
		done[v] = true
	}
	out := make([]string, 0, len(all))
	for _, v := range all {
		if !done[v] {
			out = append(out, v)
		}
	}
	return out
}
