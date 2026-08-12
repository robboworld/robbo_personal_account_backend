package usecase

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type AuthUseCaseImpl struct {
	users.Gateway
	portal                portalgateway.Gateway
	sessions              licensing.Gateway
	hashSalt              string
	accessSigningKey      []byte
	refreshSigningKey     []byte
	accessExpireDuration  time.Duration
	refreshExpireDuration time.Duration
}

type AuthUseCaseModule struct {
	fx.Out
	auth.UseCase
}

func SetupAuthUseCase(
	gateway users.Gateway,
	portal portalgateway.Gateway,
	sessions licensing.Gateway,
) AuthUseCaseModule {
	hashSalt := viper.GetString("auth.hash_salt")
	accessSigningKey := []byte(viper.GetString("auth.access_signing_key"))
	refreshSigningKey := []byte(viper.GetString("auth.refresh_signing_key"))
	accessTokenTTLTime := viper.GetDuration("auth.access_token_ttl")
	refreshTokenTTLTime := viper.GetDuration("auth.refresh_token_ttl")

	return AuthUseCaseModule{
		UseCase: &AuthUseCaseImpl{
			Gateway:               gateway,
			portal:                portal,
			sessions:              sessions,
			hashSalt:              hashSalt,
			accessSigningKey:      accessSigningKey,
			refreshSigningKey:     refreshSigningKey,
			accessExpireDuration:  accessTokenTTLTime,
			refreshExpireDuration: refreshTokenTTLTime,
		},
	}
}

func (a *AuthUseCaseImpl) SignIn(email, password string, role uint, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	if viper.GetBool("legacyPostgres.enabled") {
		return a.signInLegacy(email, password, role, client)
	}
	if auth.LmsPasswordFallbackEnabled() {
		return a.signInLMS(email, password, client)
	}
	return "", "", auth.ErrLegacyAuthDisabled
}

func (a *AuthUseCaseImpl) signInLegacy(email, password string, role uint, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	pwd := sha1.New()
	pwd.Write([]byte(password))
	pwd.Write([]byte(a.hashSalt))
	passwordHash := fmt.Sprintf("%x", pwd.Sum(nil))

	var user = new(models.UserCore)
	switch models.Role(role) {
	case models.Student:
		student, getStudentErr := a.Gateway.GetStudent(email, passwordHash)
		if getStudentErr != nil {
			return "", "", getStudentErr
		}
		user.Id = student.Id
		user.Role = models.Student
	case models.Teacher:
		teacher, getTeacherErr := a.Gateway.GetTeacher(email, passwordHash)
		if getTeacherErr != nil {
			return "", "", getTeacherErr
		}
		user.Id = teacher.Id
		user.Role = models.Teacher
	case models.Parent:
		parent, getParentErr := a.Gateway.GetParent(email, passwordHash)
		if getParentErr != nil {
			return "", "", getParentErr
		}
		user.Id = parent.Id
		user.Role = models.Parent
	case models.FreeListener:
		freeListener, getFreeListenerErr := a.Gateway.GetFreeListener(email, passwordHash)
		if getFreeListenerErr != nil {
			return "", "", getFreeListenerErr
		}
		user.Id = freeListener.Id
		user.Role = models.FreeListener
	case models.UnitAdmin:
		unitAdmin, getUnitAdminErr := a.Gateway.GetUnitAdmin(email, passwordHash)
		if getUnitAdminErr != nil {
			return "", "", getUnitAdminErr
		}
		user.Id = unitAdmin.Id
		user.Role = models.UnitAdmin
	case models.SuperAdmin:
		superAdmin, getSuperAdminErr := a.Gateway.GetSuperAdmin(email, passwordHash)
		if getSuperAdminErr != nil {
			return "", "", getSuperAdminErr
		}
		user.Id = superAdmin.Id
		user.Role = models.SuperAdmin
	default:
		err = auth.ErrUserNotFound
	}

	if err != nil {
		return "", "", err
	}

	return a.issueTokensWithSession(user, "legacy_jwt", client)
}

func (a *AuthUseCaseImpl) SignUp(userCore *models.UserCore, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	if !viper.GetBool("legacyPostgres.enabled") {
		return a.signUpLMS(userCore, client)
	}
	pwd := sha1.New()
	pwd.Write([]byte(userCore.Password))
	pwd.Write([]byte(a.hashSalt))
	userCore.Password = fmt.Sprintf("%x", pwd.Sum(nil))

	switch userCore.Role {
	case models.Student:
		student := &models.StudentCore{
			UserCore: *userCore,
		}
		newStudent, createStudentErr := a.Gateway.CreateStudent(student)
		if createStudentErr != nil {
			return "", "", createStudentErr
		}
		userCore.Id = newStudent.Id
	case models.Teacher:
		teacher := &models.TeacherCore{
			UserCore: *userCore,
		}
		newTeacher, createTeacherErr := a.Gateway.CreateTeacher(teacher)
		if createTeacherErr != nil {
			return "", "", createTeacherErr
		}
		userCore.Id = newTeacher.Id
	case models.Parent:
		parent := &models.ParentCore{
			UserCore: *userCore,
		}
		newParent, createParentErr := a.Gateway.CreateParent(parent)
		if createParentErr != nil {
			return "", "", createParentErr
		}
		userCore.Id = newParent.Id
	case models.FreeListener:
		freeListener := &models.FreeListenerCore{
			UserCore: *userCore,
		}
		newFreeListener, createFreeListenerErr := a.Gateway.CreateFreeListener(freeListener)
		if createFreeListenerErr != nil {
			return "", "", createFreeListenerErr
		}
		userCore.Id = newFreeListener.Id
	default:
		err = auth.ErrUserNotFound
	}

	if err != nil {
		return "", "", err
	}

	return a.issueTokensWithSession(userCore, "legacy_jwt", client)
}

func (a *AuthUseCaseImpl) ParseToken(token string, key []byte) (claims *models.UserClaims, err error) {
	data, err := jwt.ParseWithClaims(token, &models.UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(key), nil
		})

	if err != nil {
		return &models.UserClaims{}, err
	}

	claims, ok := data.Claims.(*models.UserClaims)
	if !ok {
		return &models.UserClaims{}, auth.ErrInvalidTypeClaims
	}
	return
}

func (a *AuthUseCaseImpl) RefreshToken(token string) (newAccessToken string, err error) {
	claims, err := a.ParseToken(token, a.refreshSigningKey)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	if claims.Sid != "" && a.sessions != nil {
		sess, sessErr := a.sessions.GetActiveSession(claims.Sid)
		if sessErr != nil || sess == nil {
			if sessErr != nil && !errors.Is(sessErr, gorm.ErrRecordNotFound) {
				log.Printf("auth refresh: get session: %v", sessErr)
			}
			return "", auth.ErrSessionNotFound
		}
		if err := a.sessions.TouchSession(claims.Sid, time.Now().UTC()); err != nil {
			log.Printf("auth refresh: touch session: %v", err)
		}
	}

	if claims.Id != "" && !lmsdb.IsUserActiveCached(claims.Id) {
		if ban := moderation.LookupPublicBanInfo(claims.Id); ban != nil {
			return "", auth.NewAccountInactiveError(ban.Reason, ban.ExpiresAt, true)
		}
		return "", auth.NewAccountInactiveError("", nil, false)
	}

	user := &models.UserCore{
		Id:   claims.Id,
		Role: claims.Role,
	}

	newAccessToken, err = a.GenerateToken(user, claims.Sid, a.accessExpireDuration, a.accessSigningKey)
	if err != nil {
		return "", err
	}

	return
}

func (a *AuthUseCaseImpl) GenerateToken(user *models.UserCore, sid string, duration time.Duration, signingKey []byte) (token string, err error) {
	claims := models.UserClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(time.Now().Add(duration * time.Second)),
		},
		Id:   user.Id,
		Role: user.Role,
		Sid:  sid,
	}
	ss := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = ss.SignedString(signingKey)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func (a *AuthUseCaseImpl) SignOut(refreshToken string) error {
	if refreshToken == "" || a.sessions == nil {
		return nil
	}
	claims, err := a.ParseToken(refreshToken, a.refreshSigningKey)
	if err != nil || claims.Sid == "" {
		return nil
	}
	return a.sessions.RevokeSession(claims.Sid)
}

func (a *AuthUseCaseImpl) ListSessions(lmsUserID string) ([]*models.UserSessionCore, error) {
	if a.sessions == nil {
		return []*models.UserSessionCore{}, nil
	}
	return a.sessions.ListActiveSessions(lmsUserID)
}

func (a *AuthUseCaseImpl) RevokeSessionByID(lmsUserID, sessionID string) error {
	if a.sessions == nil {
		return auth.ErrSessionNotFound
	}
	err := a.sessions.RevokeSessionByID(lmsUserID, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return auth.ErrSessionNotFound
	}
	return err
}

func (a *AuthUseCaseImpl) issueTokensWithSession(
	user *models.UserCore,
	authMode string,
	client auth.ClientInfo,
) (accessToken, refreshToken string, err error) {
	sid := ""
	if a.sessions != nil && user.Id != "" {
		ttl := time.Duration(a.refreshExpireDuration) * time.Second
		if ttl <= 0 {
			ttl = 7 * 24 * time.Hour
		}
		sess, createErr := licensing.AcquireLoginSession(
			a.sessions, user.Id, authMode, client.UserAgent, client.IPAddress, ttl, user.Role,
		)
		if createErr != nil {
			if errors.Is(createErr, licensing.ErrSessionLimitReached) {
				return "", "", auth.ErrSessionLimitReached
			}
			return "", "", createErr
		}
		sid = sess.SessionKey
	}

	accessToken, err = a.GenerateToken(user, sid, a.accessExpireDuration, a.accessSigningKey)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = a.GenerateToken(user, sid, a.refreshExpireDuration, a.refreshSigningKey)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
