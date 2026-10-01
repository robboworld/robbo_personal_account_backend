package db_client

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresClient struct {
	Db     *gorm.DB
	logger *log.Logger
}

// NewLogger logs slow queries and errors. Info level (every SQL statement with its bound
// values: emails, session keys, license keys) only with DEBUG=true.
func NewLogger() logger.Interface {
	level := logger.Warn
	if v := os.Getenv("DEBUG"); v == "true" || v == "1" {
		level = logger.Info
	}
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)
}

var (
	sharedMu  sync.Mutex
	sharedDBs = map[string]*gorm.DB{}
)

// OpenByDSN returns one gorm handle, and so one connection pool, per DSN for the process.
// Gateways on the same database (projects: 4, licensing: 3) each opened their own pool
// with no connection limit. A failed open is not cached, so the next call retries.
func OpenByDSN(dsn string) (*gorm.DB, error) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if db := sharedDBs[dsn]; db != nil {
		return db, nil
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: NewLogger()})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sharedDBs[dsn] = db
	return db, nil
}

// CloseAll closes the shared pools on shutdown.
func CloseAll(context.Context) error {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	var errs []error
	for dsn, db := range sharedDBs {
		if sqlDB, err := db.DB(); err == nil {
			errs = append(errs, sqlDB.Close())
		}
		delete(sharedDBs, dsn)
	}
	return errors.Join(errs...)
}

func postgresDSN() (string, error) {
	if dsn := viper.GetString("postgres.postgresDsn"); dsn != "" {
		return dsn, nil
	}
	return "", errors.New("postgres.postgresDsn required when legacyPostgres.enabled is true")
}

func NewPostgresClient(_logger *log.Logger) (postgresClient PostgresClient, err error) {
	if !viper.GetBool("legacyPostgres.enabled") {
		db, disabledErr := newDisabledDB()
		if disabledErr != nil {
			return PostgresClient{}, disabledErr
		}
		return PostgresClient{Db: db, logger: _logger}, nil
	}
	dsn, err := postgresDSN()
	if err != nil {
		return
	}
	db, err := OpenByDSN(dsn)
	if err != nil {
		return
	}
	postgresClient = PostgresClient{
		Db:     db,
		logger: _logger,
	}
	err = postgresClient.Migrate()
	return
}

// WrapDB wraps an already opened handle (test containers) without migrating it.
func WrapDB(db *gorm.DB, _logger *log.Logger) PostgresClient {
	return PostgresClient{Db: db, logger: _logger}
}

func (c *PostgresClient) Migrate() (err error) {
	if c.Db == nil {
		return nil
	}
	robboMeta := []interface{}{
		&models.CourseDB{},
		&models.CoursePacketDB{},
		&models.RobboUnitDB{},
		&models.RobboGroupDB{},
		&models.UnitAdminsRobboUnitsDB{},
		&models.TeachersRobboGroupsDB{},
		&models.StudentsOfTeacherDB{},
		&models.CourseRelationDB{},
		&models.CohortDB{},
	}
	if viper.GetBool("legacyPostgres.enabled") {
		return c.Db.AutoMigrate(append([]interface{}{
			&models.ProjectDB{},
			&models.ProjectPageDB{},
			&models.AbsoluteMediaDB{},
			&models.ImageDB{},
			&models.CourseApiMediaCollectionDB{},
			&models.MediaDB{},
			&models.TeacherDB{},
			&models.StudentDB{},
			&models.ParentDB{},
			&models.SuperAdminDB{},
			&models.UnitAdminDB{},
			&models.FreeListenerDB{},
			&models.ChildrenOfParentDB{},
		}, robboMeta...)...)
	}
	return c.Db.AutoMigrate(robboMeta...)
}
