package usecase

import (
	"log"
	"strconv"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func roleFromLMSUser(u *lmsdb.AuthUserLogin) models.Role {
	if u.IsSuperuser {
		return models.SuperAdmin
	}
	if u.IsStaff {
		return models.Teacher
	}
	return models.Student
}

func (a *AuthUseCaseImpl) signInLMS(email, password string) (accessToken, refreshToken string, err error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return "", "", err
	}
	defer reader.Close()

	u, err := reader.Authenticate(email, password)
	if err != nil {
		return "", "", err
	}

	touchLastLogin(u.ID)

	edxID := strconv.FormatInt(u.ID, 10)

	user := &models.UserCore{
		Id:    edxID,
		Email: u.Email,
		Role:  roleFromLMSUser(u),
	}

	accessToken, err = a.GenerateToken(user, a.accessExpireDuration, a.accessSigningKey)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = a.GenerateToken(user, a.refreshExpireDuration, a.refreshSigningKey)
	return accessToken, refreshToken, err
}

func touchLastLogin(userID int64) {
	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		log.Printf("lms auth: writer for last_login: %v", err)
		return
	}
	defer writer.Close()
	if err := writer.TouchLastLogin(userID); err != nil {
		log.Printf("lms auth: touch last_login for user %d: %v", userID, err)
	}
}
