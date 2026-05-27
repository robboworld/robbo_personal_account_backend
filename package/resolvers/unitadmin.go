package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// CreateUnitAdmin is the resolver for the createUnitAdmin field.
func (r *mutationResolver) CreateUnitAdmin(ctx context.Context, input models.NewUnitAdmin) (models.UnitAdminResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	unitAdminInput := models.UnitAdminHTTP{
		UserHTTP: &models.UserHTTP{
			Email:      input.Email,
			Password:   input.Password,
			Firstname:  input.Firstname,
			Lastname:   input.Lastname,
			Middlename: input.Middlename,
			Nickname:   input.Nickname,
			Role:       4,
		},
	}

	newUnitAdmin, createUnitAdminErr := r.usersDelegate.CreateUnitAdmin(&unitAdminInput)
	if createUnitAdminErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createUnitAdminErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newUnitAdmin, nil
}

// UpdateUnitAdmin is the resolver for the updateUnitAdmin field.
func (r *mutationResolver) UpdateUnitAdmin(ctx context.Context, input models.UpdateProfileInput) (models.UnitAdminResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	accessErr := profileSelfOrAdminAccess(
		ctx, ginContext, r.authDelegate, input.ID, models.UnitAdmin,
		[]models.Role{models.SuperAdmin},
	)
	if accessErr != nil {
		return nil, accessErr
	}
	if err := r.validateUsernameImmutable(ctx, input, models.UnitAdmin); err != nil {
		return nil, err
	}

	updateUnitAdminInput := &models.UnitAdminHTTP{
		UserHTTP: userHTTPFromProfileInput(input),
	}
	updateUnitAdminInput.UserHTTP.Role = 4
	unitAdminUpdated, updateUnitAdminErr := r.usersDelegate.UpdateUnitAdmin(updateUnitAdminInput)
	if updateUnitAdminErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: updateUnitAdminErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return unitAdminUpdated, nil
}

// DeleteUnitAdmin is the resolver for the deleteUnitAdmin field.
func (r *mutationResolver) DeleteUnitAdmin(ctx context.Context, unitAdminID string) (*models.DeletedUnitAdmin, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	deleteUnitAdminErr := r.usersDelegate.DeleteUnitAdmin(unitAdminID)
	if deleteUnitAdminErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteUnitAdminErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedUnitAdmin{UnitAdminID: unitAdminID}, nil
}

// SetNewUnitAdminForRobboUnit is the resolver for the setNewUnitAdminForRobboUnit field.
func (r *mutationResolver) SetNewUnitAdminForRobboUnit(ctx context.Context, unitAdminID string, robboUnitID string) (*models.Error, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	setNewUnitAdminForRobboUnitErr := r.usersDelegate.SetNewUnitAdminForRobboUnit(unitAdminID, robboUnitID)
	if setNewUnitAdminForRobboUnitErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: setNewUnitAdminForRobboUnitErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return nil, nil
}

// DeleteUnitAdminForRobboUnit is the resolver for the DeleteUnitAdminForRobboUnit field.
func (r *mutationResolver) DeleteUnitAdminForRobboUnit(ctx context.Context, unitAdminID string, robboUnitID string) (*models.Error, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	deleteUnitAdminForRobboUnitErr := r.usersDelegate.DeleteUnitAdminForRobboUnit(unitAdminID, robboUnitID)
	if deleteUnitAdminForRobboUnitErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteUnitAdminForRobboUnitErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.Error{}, nil
}

// GetAllUnitAdmins is the resolver for the GetAllUnitAdmins field.
func (r *queryResolver) GetAllUnitAdmins(ctx context.Context, page string, pageSize string) (models.UnitAdminsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	unitAdmins, countRows, getAllUnitAdminsErr := r.usersDelegate.GetAllUnitAdmins(page, pageSize)
	if getAllUnitAdminsErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllUnitAdminsErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.UnitAdminHTTPList{
		UnitAdmins: unitAdmins,
		CountRows:  countRows,
	}, nil
}

// GetUnitAdminsByRobboUnitID is the resolver for the GetUnitAdminsByRobboUnitId field.
func (r *queryResolver) GetUnitAdminsByRobboUnitID(ctx context.Context, robboUnitID string) (models.UnitAdminsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	unitAdmins, getUnitAdminByRobboUnitIdErr := r.usersDelegate.GetUnitAdminByRobboUnitId(robboUnitID)
	if getUnitAdminByRobboUnitIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getUnitAdminByRobboUnitIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.UnitAdminHTTPList{
		UnitAdmins: unitAdmins,
	}, nil
}

// GetUnitAdminByID is the resolver for the GetUnitAdminById field.
func (r *queryResolver) GetUnitAdminByID(ctx context.Context, unitAdminID string) (models.UnitAdminResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	unitAdmin, getUnitAdminByIdErr := r.usersDelegate.GetUnitAdminById(unitAdminID)
	if getUnitAdminByIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getUnitAdminByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &unitAdmin, nil
}

// SearchUnitAdminsByEmail is the resolver for the SearchUnitAdminsByEmail field.
func (r *queryResolver) SearchUnitAdminsByEmail(ctx context.Context, email string, page string, pageSize string) (models.UnitAdminsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	unitAdmins, countRows, searchUnitAdminByEmailErr := r.usersDelegate.SearchUnitAdminByEmail(email, page, pageSize)
	if searchUnitAdminByEmailErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: searchUnitAdminByEmailErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.UnitAdminHTTPList{
		UnitAdmins: unitAdmins,
		CountRows:  countRows,
	}, nil
}
