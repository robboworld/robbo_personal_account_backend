package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
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

// CreateProjectPage is the resolver for the CreateProjectPage field.
func (r *mutationResolver) CreateProjectPage(ctx context.Context) (models.ProjectPageResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	accessErr := r.authDelegate.UserAccess(userRole, projectCreateRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	newProjectPage, createProjectPageErr := r.projectPageDelegate.CreateProjectPage(
		userId,
		projectPage.LocaleFromAcceptLanguage(ginContext.GetHeader("Accept-Language")),
	)
	if createProjectPageErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createProjectPageErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &newProjectPage, nil
}

// UpdateProjectPage is the resolver for the UpdateProjectPage field.
func (r *mutationResolver) UpdateProjectPage(ctx context.Context, input models.UpdateProjectPage) (models.ProjectPageResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	accessErr := r.authDelegate.UserAccess(userRole, projectCreateRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	updateProjectPageInput := &models.ProjectPageHTTP{
		ProjectID:     input.ProjectID,
		ProjectPageID: input.ProjectPageID,
		Instruction:   input.Instruction,
		Notes:         input.Notes,
		Title:         input.Title,
		IsShared:      input.IsShared,
	}

	updateProjectPage, updateProjectPageErr := r.projectPageDelegate.UpdateProjectPage(updateProjectPageInput, userId)
	if updateProjectPageErr != nil {
		if updateProjectPageErr == auth.ErrNotAccess {
			return nil, &gqlerror.Error{
				Path:    graphql.GetPath(ctx),
				Message: updateProjectPageErr.Error(),
				Extensions: map[string]interface{}{
					"code": "403",
				},
			}
		}
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: updateProjectPageErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &updateProjectPage, nil
}

// DeleteProjectPage is the resolver for the DeleteProjectPage field.
func (r *mutationResolver) DeleteProjectPage(ctx context.Context, projectID string) (*models.DeletedProjectPage, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	accessErr := r.authDelegate.UserAccess(userRole, projectCreateRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	userId := ginContext.Value("user_id").(string)
	deleteProjectPageErr := r.projectPageDelegate.DeleteProjectPage(projectID, userId)
	if deleteProjectPageErr != nil {
		if deleteProjectPageErr == auth.ErrNotAccess {
			return nil, &gqlerror.Error{
				Path:    graphql.GetPath(ctx),
				Message: deleteProjectPageErr.Error(),
				Extensions: map[string]interface{}{
					"code": "403",
				},
			}
		}
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteProjectPageErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedProjectPage{ProjectPageID: projectID}, nil
}

// GetProjectPageByID is the resolver for the GetProjectPageById field.
func (r *queryResolver) GetProjectPageByID(ctx context.Context, projectPageID string) (models.ProjectPageResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	if err := requireAuthenticated(ctx, userRole); err != nil {
		return nil, err
	}
	accessErr := r.authDelegate.UserAccess(userRole, allAuthenticatedRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	projectPageHttp, getProjectPageByIdErr := r.projectPageDelegate.GetProjectPageById(projectPageID, userId)
	if getProjectPageByIdErr != nil {
		if getProjectPageByIdErr == auth.ErrNotAccess {
			return nil, &gqlerror.Error{
				Path:    graphql.GetPath(ctx),
				Message: getProjectPageByIdErr.Error(),
				Extensions: map[string]interface{}{
					"code": "403",
				},
			}
		}
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getProjectPageByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &projectPageHttp, nil
}

// GetPublicProjectPages is served via REST GET /projectPage/public.
// GetAllProjectPagesByUserID is the resolver for the GetAllProjectPagesByUserID field.
func (r *queryResolver) GetAllProjectPagesByUserID(ctx context.Context, userID string, page string, pageSize string) (models.ProjectPagesResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	accessErr := r.authDelegate.UserAccess(userRole, projectCreateRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	projectPages, countRows, getAllProjectPagesErr := r.projectPageDelegate.
		GetAllProjectPagesByUserId(userID, page, pageSize)
	if getAllProjectPagesErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllProjectPagesErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.ProjectPageHTTPList{
		ProjectPages: projectPages,
		CountRows:    countRows,
	}, nil
}

// GetAllProjectPagesByAccessToken is the resolver for the GetAllProjectPagesByAccessToken field.
func (r *queryResolver) GetAllProjectPagesByAccessToken(ctx context.Context, page string, pageSize string) (models.ProjectPagesResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	accessErr := r.authDelegate.UserAccess(userRole, projectCreateRoles(), ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	projectPages, countRows, getAllProjectPagesErr := r.projectPageDelegate.
		GetAllProjectPagesByUserId(userId, page, pageSize)
	if getAllProjectPagesErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllProjectPagesErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.ProjectPageHTTPList{
		ProjectPages: projectPages,
		CountRows:    countRows,
	}, nil
}
