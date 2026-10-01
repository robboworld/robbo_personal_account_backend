package resolvers

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func allAuthenticatedRoles() []models.Role {
	return []models.Role{
		models.Student,
		models.Teacher,
		models.Parent,
		models.FreeListener,
		models.UnitAdmin,
		models.SuperAdmin,
	}
}

func projectCreateRoles() []models.Role {
	return []models.Role{models.Student, models.UnitAdmin, models.SuperAdmin}
}

func requireAuthenticated(ctx context.Context, userRole models.Role) error {
	if userRole == models.Anonymous {
		return &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: auth.ErrTokenNotFound.Error(),
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
	}
	return nil
}

func signInErrorCode(err error) string {
	switch {
	case errors.Is(err, auth.ErrUserNotFound), errors.Is(err, auth.ErrInvalidCredentials):
		return "401"
	case errors.Is(err, auth.ErrUserInactive):
		return "403"
	case errors.Is(err, auth.ErrLegacyAuthDisabled):
		return "410"
	case errors.Is(err, auth.ErrSessionLimitReached):
		return "SESSION_LIMIT_REACHED"
	default:
		return "500"
	}
}

func getRefreshToken(c *gin.Context) (refreshToken string, err error) {
	refreshToken, _ = c.Value("refresh_token").(string)
	if refreshToken == "" {
		return "", errors.New("error finding cookie")
	}
	return
}
func setRefreshToken(value string, c *gin.Context) {
	if value == "" {
		oidc.ClearHTTPOnlyCookie(c, "refresh_token")
		return
	}
	oidc.SetHTTPOnlyCookie(c, "refresh_token", value, 60*60*24*7)
}
