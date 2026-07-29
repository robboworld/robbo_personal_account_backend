package usecase

import (
	"log"
	"strconv"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
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

func (a *AuthUseCaseImpl) signInLMS(email, password string, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return "", "", err
	}
	defer reader.Close()

	u, err := reader.LookupAuthUserForLogin(email)
	if err != nil {
		return "", "", err
	}
	if u == nil {
		return "", "", auth.ErrUserNotFound
	}
	if !u.IsActive {
		edxID := strconv.FormatInt(u.ID, 10)
		if ban := moderation.LookupPublicBanInfo(edxID); ban != nil {
			return "", "", auth.NewAccountInactiveError(ban.Reason, ban.ExpiresAt, true)
		}
		return "", "", auth.NewAccountInactiveError("", nil, false)
	}
	if !lmsdb.VerifyDjangoPassword(password, u.Password) {
		return "", "", auth.ErrInvalidCredentials
	}

	touchLastLogin(u.ID)

	edxID := strconv.FormatInt(u.ID, 10)

	user := &models.UserCore{
		Id:    edxID,
		Email: u.Email,
		Role:  roleFromLMSUser(u),
	}

	return a.issueTokensWithSession(user, "lms_db", client)
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
