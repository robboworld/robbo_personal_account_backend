package db_client

import (
	"github.com/ory/dockertest/v3"
	"log"
	"os"
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

func NewLogger() logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)
}

func OpenByDSN(dsn string) (db *gorm.DB, err error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: NewLogger()})
}

func postgresDSN() string {
	if !viper.GetBool("legacyPostgres.enabled") {
		return ""
	}
	if dsn := viper.GetString("postgres.postgresDsn"); dsn != "" {
		return dsn
	}
	panic("postgres.postgresDsn required when legacyPostgres.enabled is true")
}

func NewPostgresClient(_logger *log.Logger) (postgresClient PostgresClient, err error) {
	if !viper.GetBool("legacyPostgres.enabled") {
		return PostgresClient{Db: nil, logger: _logger}, nil
	}
	db, err := OpenByDSN(postgresDSN())
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

func NewTestPostgresClient(_logger *log.Logger, testDockerClient dockertest.Pool) (testPostgresClient PostgresClient, err error) {
	var gdb *gorm.DB
	if err = testDockerClient.Retry(func() error {
		gdb, err = OpenByDSN(viper.GetString("postgres.postgresDsn"))
		if err != nil {
			log.Println("Test database not ready yet (it is booting up, wait for a few tries)...")
			return err
		}
		db, sqlErr := gdb.DB()
		if sqlErr != nil {
			return sqlErr
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	testPostgresClient = PostgresClient{
		Db:     gdb,
		logger: _logger,
	}
	err = testPostgresClient.Migrate()
	return
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
