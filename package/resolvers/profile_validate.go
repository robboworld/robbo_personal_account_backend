package resolvers

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func usernameChangeForbidden(ctx context.Context, currentUsername, requested string) error {
	if strings.TrimSpace(currentUsername) == strings.TrimSpace(requested) {
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

func (r *mutationResolver) validateUsernameImmutable(
	ctx context.Context,
	input models.UpdateProfileInput,
	role models.Role,
) error {
	var current string

	switch role {
	case models.Student:
		u, e := r.usersDelegate.GetStudentById(input.ID)
		if e != nil {
			return e
		}
		current = u.UserHTTP.Nickname
	case models.Teacher:
		u, e := r.usersDelegate.GetTeacherById(input.ID)
		if e != nil {
			return e
		}
		current = u.UserHTTP.Nickname
	case models.Parent:
		u, e := r.usersDelegate.GetParentById(input.ID)
		if e != nil {
			return e
		}
		current = u.UserHTTP.Nickname
	case models.UnitAdmin:
		u, e := r.usersDelegate.GetUnitAdminById(input.ID)
		if e != nil {
			return e
		}
		current = u.UserHTTP.Nickname
	case models.SuperAdmin:
		u, e := r.usersDelegate.GetSuperAdminById(input.ID)
		if e != nil {
			return e
		}
		current = u.UserHTTP.Nickname
	case models.FreeListener:
		u, e := r.usersDelegate.GetFreeListenerById(input.ID)
		if e != nil {
			return e
		}
		current = u.Nickname
	default:
		return nil
	}
	return usernameChangeForbidden(ctx, current, input.Nickname)
}
