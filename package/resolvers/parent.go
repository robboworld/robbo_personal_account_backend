package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// CreateParent is the resolver for the createParent field.
func (r *mutationResolver) CreateParent(ctx context.Context, input models.NewParent) (models.ParentResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	parentInput := models.ParentHTTP{
		UserHTTP: &models.UserHTTP{
			Email:      input.Email,
			Password:   input.Password,
			Firstname:  input.Firstname,
			Lastname:   input.Lastname,
			Middlename: input.Middlename,
			Nickname:   input.Nickname,
			Role:       2,
		},
		Children: nil,
	}
	newParent, createParentErr := r.usersDelegate.CreateParent(&parentInput)
	if createParentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createParentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newParent, nil
}

// UpdateParent is the resolver for the updateParent field.
func (r *mutationResolver) UpdateParent(ctx context.Context, input models.UpdateProfileInput) (models.ParentResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	accessErr := profileSelfOrAdminAccess(
		ctx, ginContext, r.authDelegate, input.ID, models.Parent,
		[]models.Role{models.UnitAdmin, models.SuperAdmin},
	)
	if accessErr != nil {
		return nil, accessErr
	}
	if err := r.validateUsernameImmutable(ctx, input, models.Parent); err != nil {
		return nil, err
	}

	updateParentInput := &models.ParentHTTP{
		UserHTTP: userHTTPFromProfileInput(input),
	}
	updateParentInput.UserHTTP.Role = 2
	parentUpdated, updateParentErr := r.usersDelegate.UpdateParent(updateParentInput)
	if updateParentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: updateParentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return parentUpdated, nil
}

// AddChildToParent is the resolver for the addChildToParent field.
func (r *mutationResolver) AddChildToParent(ctx context.Context, parentID string, childID string) (models.StudentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	students, createRelationErr := r.usersDelegate.CreateStudentParentRelation(parentID, childID)
	if createRelationErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createRelationErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{Students: students}, nil
}

// DeleteParent is the resolver for the deleteParent field.
func (r *mutationResolver) DeleteParent(ctx context.Context, parentID string) (*models.DeletedParent, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	deleteParentErr := r.usersDelegate.DeleteParent(parentID)
	if deleteParentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteParentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedParent{ParentID: parentID}, nil
}

// GetAllParents is the resolver for the GetAllParents field.
func (r *queryResolver) GetAllParents(ctx context.Context, page string, pageSize string) (models.ParentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	parents, countRows, getAllParentsErr := r.usersDelegate.GetAllParent(page, pageSize)
	if getAllParentsErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllParentsErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.ParentHTTPList{
		Parents:   parents,
		CountRows: countRows,
	}, nil
}

// GetParentByID is the resolver for the GetParentById field.
func (r *queryResolver) GetParentByID(ctx context.Context, parentID string) (models.ParentResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.Parent, models.Teacher, models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	parent, getParentByIdErr := r.usersDelegate.GetParentById(parentID)
	if getParentByIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getParentByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return parent, nil
}

// GetPairsStudentParentsByAccessToken is the resolver for the GetPairsStudentParentsByAccessToken field.
func (r *queryResolver) GetPairsStudentParentsByAccessToken(ctx context.Context) (models.PairsStudentParentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin, models.Teacher}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	pairsStudentParents, getPairsStudentParentsByAccessTokenErr := r.usersDelegate.
		GetPairsStudentParentsByTeacherId(userId)
	if getPairsStudentParentsByAccessTokenErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getPairsStudentParentsByAccessTokenErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentParentsHTTPList{
		PairsStudentParents: pairsStudentParents,
	}, nil
}
