package db_client

import (
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

func NewPostgresClient(_logger *log.Logger) (postgresClient PostgresClient, err error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)
	db, err := gorm.Open(postgres.Open(viper.GetString("postgres.postgresDsn")), &gorm.Config{Logger: newLogger})
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

func (c *PostgresClient) Migrate() (err error) {
	err = c.Db.AutoMigrate(
		&models.ProjectDB{},
		&models.ProjectPageDB{},
		&models.CourseDB{},
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
		&models.RobboUnitDB{},
		&models.CoursePacketDB{},
		&models.RobboGroupDB{},
		&models.UnitAdminsRobboUnitsDB{},
		&models.TeachersRobboGroupsDB{},
	)
	return
}
