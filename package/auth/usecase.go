package auth

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// ClientInfo carries request metadata stored on lk_user_sessions rows.
type ClientInfo struct {
	UserAgent string
	IPAddress string
}

type UseCase interface {
	SignIn(email, password string, role uint, client ClientInfo) (accessToken string, refreshToken string, err error)
	SignUp(userCore *models.UserCore, client ClientInfo) (accessToken string, refreshToken string, err error)
	ParseToken(token string, key []byte) (claims *models.UserClaims, err error)
	RefreshToken(refreshToken string) (newAccessToken string, err error)
	GenerateToken(user *models.UserCore, sid string, duration time.Duration, signingKey []byte) (token string, err error)
	// SignOut best-effort revokes the session tied to refreshToken (no-op on parse errors).
	SignOut(refreshToken string) error
	ListSessions(lmsUserID string) ([]*models.UserSessionCore, error)
	RevokeSessionByID(lmsUserID, sessionID string) error
}
