package resolvers

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users/profileaccess"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// profileReadAccess rejects reading someone else's profile by ID unless profileaccess allows it.
func profileReadAccess(ctx context.Context, c *gin.Context, role models.Role, targetID string) error {
	viewerID, _ := c.Value("user_id").(string)
	if profileaccess.CanRead(viewerID, role, targetID) {
		return nil
	}
	return &gqlerror.Error{
		Path:    graphql.GetPath(ctx),
		Message: auth.ErrNotAccess.Error(),
		Extensions: map[string]interface{}{
			"code": "403",
		},
	}
}

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
	return []models.Role{models.Student, models.Teacher, models.UnitAdmin, models.SuperAdmin}
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
