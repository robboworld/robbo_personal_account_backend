package usecase

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/streak"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type AuthUseCaseImpl struct {
	users.Gateway
	portal                portalgateway.Gateway
	sessions              licensing.Gateway
	streak                streak.UseCase
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
	streakUC streak.UseCase,
) AuthUseCaseModule {
	accessSigningKey := []byte(viper.GetString("auth.access_signing_key"))
	refreshSigningKey := []byte(viper.GetString("auth.refresh_signing_key"))
	accessTokenTTLTime := viper.GetDuration("auth.access_token_ttl")
	refreshTokenTTLTime := viper.GetDuration("auth.refresh_token_ttl")

	return AuthUseCaseModule{
		UseCase: &AuthUseCaseImpl{
			Gateway:               gateway,
			portal:                portal,
			sessions:              sessions,
			streak:                streakUC,
			accessSigningKey:      accessSigningKey,
			refreshSigningKey:     refreshSigningKey,
			accessExpireDuration:  accessTokenTTLTime,
			refreshExpireDuration: refreshTokenTTLTime,
		},
	}
}

// SignIn checks email/password against LMS (lms_db mode or the LMS password fallback).
// The SHA1 login against the legacy Postgres users tables was removed.
func (a *AuthUseCaseImpl) SignIn(email, password string, role uint, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	if auth.LmsPasswordFallbackEnabled() {
		return a.signInLMS(email, password, client)
	}
	return "", "", auth.ErrLegacyAuthDisabled
}

// SignUp registers the account in LMS; the legacy Postgres sign-up was removed.
func (a *AuthUseCaseImpl) SignUp(userCore *models.UserCore, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	return a.signUpLMS(userCore, client)
}

func (a *AuthUseCaseImpl) ParseToken(token string, key []byte) (claims *models.UserClaims, err error) {
	data, err := jwt.ParseWithClaims(token, &models.UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(key), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return &models.UserClaims{}, err
	}

	claims, ok := data.Claims.(*models.UserClaims)
	if !ok {
		return &models.UserClaims{}, auth.ErrInvalidTypeClaims
	}
	return
}

func (a *AuthUseCaseImpl) RefreshToken(token string) (newAccessToken string, newRefreshToken string, err error) {
	claims, err := a.ParseToken(token, a.refreshSigningKey)
	if err != nil {
		log.Println(err)
		return "", "", err
	}

	if claims.Sid == "" || a.sessions == nil {
		return "", "", auth.ErrSessionNotFound
	}
	sess, sessErr := a.sessions.GetActiveSession(claims.Sid)
	if sessErr != nil || sess == nil {
		if sessErr != nil && !errors.Is(sessErr, gorm.ErrRecordNotFound) {
			log.Printf("auth refresh: get session: %v", sessErr)
		}
		return "", "", auth.ErrSessionNotFound
	}
	if err := a.sessions.TouchSession(claims.Sid, time.Now().UTC()); err != nil {
		log.Printf("auth refresh: touch session: %v", err)
	}

	if claims.Id != "" {
		active, activeErr := lmsdb.IsUserActiveCached(claims.Id)
		if activeErr != nil {
			return "", "", fmt.Errorf("auth refresh: lms unavailable: %w", activeErr)
		}
		if !active {
			if ban := moderation.LookupPublicBanInfo(claims.Id); ban != nil {
				return "", "", auth.NewAccountInactiveError(ban.Reason, ban.ExpiresAt, true)
			}
			return "", "", auth.NewAccountInactiveError("", nil, false)
		}
	}

	user := &models.UserCore{
		Id:   claims.Id,
		Role: claims.Role,
	}

	newAccessToken, err = a.GenerateToken(user, claims.Sid, a.accessExpireDuration, a.accessSigningKey)
	if err != nil {
		return "", "", err
	}
	newRefreshToken, err = a.GenerateToken(user, claims.Sid, a.refreshExpireDuration, a.refreshSigningKey)
	if err != nil {
		return "", "", err
	}

	return
}

func (a *AuthUseCaseImpl) GenerateToken(user *models.UserCore, sid string, duration time.Duration, signingKey []byte) (token string, err error) {
	claims := models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration * time.Second)),
		},
		Id:   user.Id,
		Role: user.Role,
		Sid:  sid,
	}
	ss := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = ss.SignedString(signingKey)
	if err != nil {
		log.Println(err)
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
		if client.KickOtherSessions {
			if kickErr := licensing.KickOtherSessions(a.sessions, user.Id); kickErr != nil {
				return "", "", kickErr
			}
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
