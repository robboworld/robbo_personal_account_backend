package usecase

import (
	"log"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func resolveFullName(core *models.UserCore) string {
	if core == nil {
		return ""
	}
	if name := strings.TrimSpace(core.FullName); name != "" {
		return name
	}
	parts := []string{strings.TrimSpace(core.Firstname), strings.TrimSpace(core.Lastname)}
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " ")
}

func (a *AuthUseCaseImpl) signUpLMS(userCore *models.UserCore) (accessToken, refreshToken string, err error) {
	if userCore == nil {
		return "", "", auth.ErrUserNotFound
	}
	fullName := resolveFullName(userCore)
	username := strings.TrimSpace(userCore.Nickname)
	email := strings.TrimSpace(userCore.Email)
	company := strings.TrimSpace(userCore.Company)
	password := userCore.Password
	if username == "" || email == "" || password == "" {
		return "", "", auth.ErrUserNotFound
	}
	if company == "" {
		return "", "", auth.ErrCompanyRequired
	}

	encoded, err := lmsdb.EncodeDjangoPassword(password)
	if err != nil {
		return "", "", err
	}

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return "", "", err
	}
	defer writer.Close()

	userID, err := writer.CreateUserWithProfile(username, email, encoded, fullName, company)
	if err != nil {
		return "", "", err
	}

	if err := writer.TouchLastLogin(userID); err != nil {
		log.Printf("lms sign-up: touch last_login for user %d: %v", userID, err)
	}

	userCore.Id = strconv.FormatInt(userID, 10)
	userCore.Role = models.Student
	userCore.FullName = fullName

	accessToken, err = a.GenerateToken(userCore, a.accessExpireDuration, a.accessSigningKey)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = a.GenerateToken(userCore, a.refreshExpireDuration, a.refreshSigningKey)
	return accessToken, refreshToken, err
}
