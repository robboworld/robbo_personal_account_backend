package usecase

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	emailPattern    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	ruPhonePattern = regexp.MustCompile(`^\+7\d{10}$`)
)

func isValidInternationalPhone(phone string) bool {
	if !strings.HasPrefix(phone, "+") || strings.HasPrefix(phone, "+7") {
		return false
	}
	digits := phone[1:]
	if len(digits) < 8 || len(digits) > 15 || digits[0] == '0' {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

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

func nameHasThreeWords(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	return len(strings.Fields(trimmed)) == 3
}

func isValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func isValidPhoneNumber(phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return true
	}
	return ruPhonePattern.MatchString(phone) || isValidInternationalPhone(phone)
}

func (a *AuthUseCaseImpl) signUpLMS(userCore *models.UserCore) (accessToken, refreshToken string, err error) {
	if userCore == nil {
		return "", "", auth.ErrUserNotFound
	}
	fullName := resolveFullName(userCore)
	username := strings.TrimSpace(userCore.Nickname)
	email := strings.TrimSpace(userCore.Email)
	phone := strings.TrimSpace(userCore.PhoneNumber)
	password := userCore.Password

	if username == "" || email == "" || password == "" || fullName == "" {
		return "", "", auth.ErrInvalidRegistration
	}
	if !userCore.HonorCode {
		return "", "", auth.ErrHonorCodeRequired
	}
	if !userCore.MarketingOptIn {
		return "", "", auth.ErrMarketingOptInRequired
	}
	if !nameHasThreeWords(fullName) {
		return "", "", auth.ErrInvalidRegistration
	}
	if !emailPattern.MatchString(email) {
		return "", "", auth.ErrInvalidRegistration
	}
	if len(username) < 2 || len(username) > 30 || !usernamePattern.MatchString(username) {
		return "", "", auth.ErrInvalidRegistration
	}
	if !isValidPassword(password) {
		return "", "", auth.ErrInvalidRegistration
	}
	if !isValidPhoneNumber(phone) {
		return "", "", auth.ErrInvalidRegistration
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

	userID, err := writer.CreateUserWithProfile(username, email, encoded, fullName, phone, userCore.MarketingOptIn)
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
