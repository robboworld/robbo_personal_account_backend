package gateway

import (
	"errors"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

func portalLinkToStudentCore(link *models.RobboPortalUserLinkDB) *models.StudentCore {
	id := ""
	if link.EdxUserID != nil {
		id = *link.EdxUserID
	} else if link.LegacyLKUserID != nil {
		id = strings.TrimSpace(*link.LegacyLKUserID)
	}
	name := ""
	if link.DisplayName != nil {
		name = *link.DisplayName
	}
	return &models.StudentCore{
		UserCore: models.UserCore{
			Id:       id,
			Email:    link.Email,
			Nickname: name,
			Role:     models.Student,
		},
	}
}

func (r *UsersGatewayImpl) searchPortalStudentsByEmail(email string, page, pageSize int) (
	students []*models.StudentCore,
	countRows int64,
	err error,
) {
	offset := (page - 1) * pageSize
	pattern := "%" + strings.TrimSpace(email) + "%"
	var links []models.RobboPortalUserLinkDB
	q := r.PostgresClient.Db.Model(&models.RobboPortalUserLinkDB{}).
		Where("lower(email) LIKE lower(?)", pattern)
	if err = q.Count(&countRows).Error; err != nil {
		return
	}
	err = q.Order("id ASC").Limit(pageSize).Offset(offset).Find(&links).Error
	if err != nil {
		return
	}
	for i := range links {
		students = append(students, portalLinkToStudentCore(&links[i]))
	}
	return
}

func (r *UsersGatewayImpl) findPortalStudentByEmail(email string) (*models.StudentCore, error) {
	var link models.RobboPortalUserLinkDB
	err := r.PostgresClient.Db.
		Where("lower(email) = ?", strings.TrimSpace(strings.ToLower(email))).
		First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return portalLinkToStudentCore(&link), nil
}
