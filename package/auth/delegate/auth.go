package delegate

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
)

const (
	authorizationHeader = "Authorization"
)

type AuthDelegateImpl struct {
	auth.UseCase
}

type AuthDelegateModule struct {
	fx.Out
	auth.Delegate
}

func SetupAuthDelegate(usecase auth.UseCase) AuthDelegateModule {
	return AuthDelegateModule{
		Delegate: &AuthDelegateImpl{usecase},
	}
}

func (s *AuthDelegateImpl) SignIn(email, password string, role uint, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	return s.UseCase.SignIn(email, password, role, client)
}

func (s *AuthDelegateImpl) SignUp(userHttp *models.UserHTTP, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	userCore := userHttp.ToCore()
	return s.UseCase.SignUp(&userCore, client)
}

func (s *AuthDelegateImpl) SignUpCore(userCore *models.UserCore, client auth.ClientInfo) (accessToken, refreshToken string, err error) {
	return s.UseCase.SignUp(userCore, client)
}

func (s *AuthDelegateImpl) RefreshToken(refreshToken string) (newAccessToken string, newRefreshToken string, err error) {
	return s.UseCase.RefreshToken(refreshToken)
}

func (s *AuthDelegateImpl) SignOut(refreshToken string) error {
	return s.UseCase.SignOut(refreshToken)
}

func (s *AuthDelegateImpl) ListSessions(lmsUserID string) ([]*models.UserSessionCore, error) {
	return s.UseCase.ListSessions(lmsUserID)
}

func (s *AuthDelegateImpl) RevokeSessionByID(lmsUserID, sessionID string) error {
	return s.UseCase.RevokeSessionByID(lmsUserID, sessionID)
}

func unauthorizedIdentity(c *gin.Context, message string) (string, models.Role, error) {
	return "", models.Anonymous, &gqlerror.Error{
		Path:    graphql.GetPath(c),
		Message: message,
		Extensions: map[string]interface{}{
			"code": "401",
		},
	}
}

func (s *AuthDelegateImpl) UserIdentity(c *gin.Context) (id string, role models.Role, err error) {
	// Wherever TokenAuthMiddleware ran, its verdict is final: it has already checked the
	// session row and the account. Re-parsing the header here would let a revoked or banned
	// token through, since the middleware marks such a request Anonymous and moves on.
	if rawRole, ok := c.Get("user_role"); ok {
		role, _ = rawRole.(models.Role)
		uid := ""
		if rawID, ok := c.Get("user_id"); ok {
			uid, _ = rawID.(string)
			uid = strings.TrimSpace(uid)
		}
		if uid != "" && uid != "0" && role != models.Anonymous && role != models.WithExpiredToken {
			return uid, role, nil
		}
		return unauthorizedIdentity(c, auth.ErrTokenNotFound.Error())
	}

	// Paths the middleware skips (public routes): legacy JWT Bearer only.
	header := c.GetHeader(authorizationHeader)
	if header == "" {
		return unauthorizedIdentity(c, auth.ErrTokenNotFound.Error())
	}

	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 {
		return unauthorizedIdentity(c, auth.ErrTokenNotFound.Error())
	}

	claims, parseTokenErr := s.UseCase.ParseToken(headerParts[1], []byte(viper.GetString("auth.access_signing_key")))
	if parseTokenErr != nil {
		return unauthorizedIdentity(c, parseTokenErr.Error())
	}
	// A BFF session token shares the signing key but has no Id: it names nobody here.
	if strings.TrimSpace(claims.Id) == "" {
		return unauthorizedIdentity(c, auth.ErrTokenNotFound.Error())
	}
	return claims.Id, claims.Role, nil
}

func (s *AuthDelegateImpl) UserAccess(currentRole models.Role, roles []models.Role, ctx context.Context) (err error) {
	for _, role := range roles {
		if currentRole == role {
			return nil
		}
	}
	err = &gqlerror.Error{
		Path:    graphql.GetPath(ctx),
		Message: auth.ErrNotAccess.Error(),
		Extensions: map[string]interface{}{
			"code": "403",
		},
	}
	return
}
