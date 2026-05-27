package gateway

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func lmsRoleFromProfile(u *lmsdb.AuthUserProfile) models.Role {
	if u == nil {
		return models.Student
	}
	if u.IsSuperuser {
		return models.SuperAdmin
	}
	if u.IsStaff {
		return models.Teacher
	}
	return models.Student
}

func userCoreFromLMSProfile(u *lmsdb.AuthUserProfile) models.UserCore {
	if u == nil {
		return models.UserCore{}
	}
	return models.UserCore{
		Id:         strconv.FormatInt(u.ID, 10),
		Email:      u.Email,
		Nickname:   strings.TrimSpace(u.Username),
		Firstname:  strings.TrimSpace(u.FirstName),
		Lastname:   strings.TrimSpace(u.LastName),
		Middlename: "",
		Role:       lmsRoleFromProfile(u),
		CreatedAt:  u.DateJoined.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (r *UsersGatewayImpl) lmsUserCoreByID(userID string) (models.UserCore, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil || id <= 0 {
		return models.UserCore{}, auth.ErrUserNotFound
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return models.UserCore{}, err
	}
	defer reader.Close()

	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil {
		return models.UserCore{}, err
	}
	if profile == nil {
		return models.UserCore{}, auth.ErrUserNotFound
	}
	return userCoreFromLMSProfile(profile), nil
}

func (r *UsersGatewayImpl) updateLMSUserProfile(userID string, core models.UserCore) (models.UserCore, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil || id <= 0 {
		return models.UserCore{}, auth.ErrUserNotFound
	}
	email := strings.TrimSpace(core.Email)
	if email == "" {
		return models.UserCore{}, auth.ErrUserNotFound
	}

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return models.UserCore{}, err
	}
	defer writer.Close()

	if err := writer.UpdateProfile(id, email, strings.TrimSpace(core.Firstname), strings.TrimSpace(core.Lastname)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UserCore{}, auth.ErrUserNotFound
		}
		return models.UserCore{}, err
	}
	return r.lmsUserCoreByID(userID)
}

func (r *UsersGatewayImpl) searchLMSStudentsByEmail(email string, page, pageSize int) (
	students []*models.StudentCore,
	countRows int64,
	err error,
) {
	offset := (page - 1) * pageSize
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	profiles, count, err := reader.SearchAuthUserProfilesByEmail(strings.TrimSpace(email), pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	countRows = count
	for i := range profiles {
		core := userCoreFromLMSProfile(&profiles[i])
		students = append(students, &models.StudentCore{UserCore: core})
	}
	return students, countRows, nil
}
