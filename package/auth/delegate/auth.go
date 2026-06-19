package delegate

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
	"strings"
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

func (s *AuthDelegateImpl) SignIn(email, password string, role uint) (accessToken, refreshToken string, err error) {
	return s.UseCase.SignIn(email, password, role)
}

func (s *AuthDelegateImpl) SignUp(userHttp *models.UserHTTP) (accessToken, refreshToken string, err error) {
	userCore := userHttp.ToCore()
	return s.UseCase.SignUp(&userCore)
}

func (s *AuthDelegateImpl) SignUpCore(userCore *models.UserCore) (accessToken, refreshToken string, err error) {
	return s.UseCase.SignUp(userCore)
}

func (s *AuthDelegateImpl) RefreshToken(refreshToken string) (newAccessToken string, err error) {
	return s.UseCase.RefreshToken(refreshToken)
}

func (s *AuthDelegateImpl) UserIdentity(c *gin.Context) (id string, role models.Role, err error) {
	header := c.GetHeader(authorizationHeader)
	if header == "" {
		return "", models.Anonymous, &gqlerror.Error{
			Path:    graphql.GetPath(c),
			Message: auth.ErrTokenNotFound.Error(),
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
	}

	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 {
		return "", models.Anonymous, &gqlerror.Error{
			Path:    graphql.GetPath(c),
			Message: auth.ErrTokenNotFound.Error(),
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
	}

	claims, parseTokenErr := s.UseCase.ParseToken(headerParts[1], []byte(viper.GetString("auth.access_signing_key")))
	if parseTokenErr != nil {
		return "", models.Anonymous, &gqlerror.Error{
			Path:    graphql.GetPath(c),
			Message: parseTokenErr.Error(),
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
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
