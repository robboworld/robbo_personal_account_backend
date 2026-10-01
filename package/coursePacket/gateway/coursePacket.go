package gateway

import (
	"log"
	"strconv"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type CoursePacketGatewayImpl struct {
	PostgresClient *db_client.PostgresClient
}

type CoursePacketGatewayModule struct {
	fx.Out
	coursePacket.Gateway
}

func SetupCoursePacketGateway(postgresClient db_client.PostgresClient) CoursePacketGatewayModule {
	return CoursePacketGatewayModule{
		Gateway: &CoursePacketGatewayImpl{PostgresClient: &postgresClient},
	}
}

func (r *CoursePacketGatewayImpl) CreateCoursePacket(coursePacketCore *models.CoursePacketCore) (id string, err error) {
	coursePacketDb := models.CoursePacketDB{}
	coursePacketDb.FromCore(coursePacketCore)

	err = r.PostgresClient.Db.Transaction(func(tx *gorm.DB) (err error) {
		err = tx.Create(&coursePacketDb).Error
		if err != nil {
			log.Println(err)
			return
		}
		return
	})
	if err != nil {
		log.Println(err)
		return "", err
	}
	id = strconv.FormatUint(uint64(coursePacketDb.ID), 10)
	return id, nil
}

func (r CoursePacketGatewayImpl) DeleteCoursePacket(coursePacketId string) (id string, err error) {
	crsPacket := models.CoursePacketDB{}
	err = r.PostgresClient.Db.Transaction(func(tx *gorm.DB) (err error) {
		err = tx.Model(&crsPacket).Where("id = ?", coursePacketId).First(&models.CoursePacketDB{}).Delete(&models.CoursePacketDB{}).Error
		if err != nil {
			err = coursePacket.ErrCoursePacketNotFound
			log.Println(err)
			return
		}
		return
	})
	if err != nil {
		log.Println(err)
		return
	}
	id = strconv.FormatUint(uint64(crsPacket.ID), 10)
	return
}

func (r *CoursePacketGatewayImpl) UpdateCoursePacket(coursePacketCore *models.CoursePacketCore) (err error) {
	coursePacketDb := models.CoursePacketDB{}
	coursePacketDb.FromCore(coursePacketCore)
	err = r.PostgresClient.Db.Transaction(func(tx *gorm.DB) (err error) {
		err = tx.Model(&coursePacketDb).Where("ID = ?", coursePacketDb.ID).First(&models.CoursePacketDB{}).Updates(coursePacketDb).Error
		if err != nil {
			err = coursePacket.ErrCoursePacketNotFound
			log.Println(err)
			return
		}
		return
	})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func (r *CoursePacketGatewayImpl) GetAllCoursePackets() (coursePacketsCore []*models.CoursePacketCore, err error) {
	var coursePacketsDB []*models.CoursePacketDB
	err = r.PostgresClient.Db.Transaction(func(tx *gorm.DB) (err error) {
		if err = tx.Find(&coursePacketsDB).Error; err != nil {
			return
		}
		return
	})

	for _, coursePacketDB := range coursePacketsDB {
		coursePacketsCore = append(coursePacketsCore, coursePacketDB.ToCore())
	}
	return
}

func (r *CoursePacketGatewayImpl) GetCoursePacketById(coursePacketId string) (coursePacketCore *models.CoursePacketCore, err error) {
	var coursePacketDb models.CoursePacketDB

	err = r.PostgresClient.Db.Transaction(func(tx *gorm.DB) (err error) {
		if err = tx.Where("id = ?", coursePacketId).First(&coursePacketDb).Error; err != nil {
			err = coursePacket.ErrCoursePacketNotFound
			log.Println(err)
			return
		}
		return
	})
	coursePacketCore = coursePacketDb.ToCore()
	return
}
