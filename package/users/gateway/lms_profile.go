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

func nullStringToValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullIntToPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

func userCoreFromLMSProfile(u *lmsdb.AuthUserProfile) models.UserCore {
	if u == nil {
		return models.UserCore{}
	}
	fullName := strings.TrimSpace(u.ProfileName)
	if fullName == "" {
		parts := []string{strings.TrimSpace(u.FirstName), strings.TrimSpace(u.LastName)}
		var nonEmpty []string
		for _, p := range parts {
			if p != "" {
				nonEmpty = append(nonEmpty, p)
			}
		}
		fullName = strings.Join(nonEmpty, " ")
	}
	return models.UserCore{
		Id:               strconv.FormatInt(u.ID, 10),
		Email:            u.Email,
		Nickname:         strings.TrimSpace(u.Username),
		FullName:         fullName,
		Firstname:        "",
		Lastname:         "",
		Middlename:       "",
		LevelOfEducation: nullStringToValue(u.LevelOfEducation),
		Country:          nullStringToValue(u.Country),
		YearOfBirth:      nullIntToPtr(u.YearOfBirth),
		Gender:           nullStringToValue(u.Gender),
		Language:         nullStringToValue(u.Language),
		Bio:              nullStringToValue(u.Bio),
		Role:             lmsRoleFromProfile(u),
		CreatedAt:        u.DateJoined.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func profileExtendedFromCore(core models.UserCore) lmsdb.ProfileExtendedUpdate {
	return lmsdb.ProfileExtendedUpdate{
		LevelOfEducation: core.LevelOfEducation,
		Country:          core.Country,
		YearOfBirth:      core.YearOfBirth,
		Gender:           core.Gender,
		Language:         core.Language,
		Bio:              core.Bio,
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
	fullName := strings.TrimSpace(core.FullName)

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return models.UserCore{}, err
	}
	defer writer.Close()

	if err := writer.UpdateProfile(id, email, fullName, profileExtendedFromCore(core)); err != nil {
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
