package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// CreateTeacher is the resolver for the createTeacher field.
func (r *mutationResolver) CreateTeacher(ctx context.Context, input models.NewTeacher) (models.TeacherResult, error) {
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

	teacherInput := models.TeacherHTTP{
		UserHTTP: &models.UserHTTP{
			Email:      input.Email,
			Password:   input.Password,
			Firstname:  input.Firstname,
			Lastname:   input.Lastname,
			Middlename: input.Middlename,
			Nickname:   input.Nickname,
			Role:       1,
		},
	}

	newTeacher, createTeacherErr := r.usersDelegate.CreateTeacher(&teacherInput)
	if createTeacherErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createTeacherErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}

	return &newTeacher, nil
}

// UpdateTeacher is the resolver for the updateTeacher field.
func (r *mutationResolver) UpdateTeacher(ctx context.Context, input models.UpdateProfileInput) (models.TeacherResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	accessErr := profileSelfOrAdminAccess(
		ctx, ginContext, r.authDelegate, input.ID, models.Teacher,
		[]models.Role{models.UnitAdmin, models.SuperAdmin},
	)
	if accessErr != nil {
		return nil, accessErr
	}
	if err := r.validateUsernameImmutable(ctx, input, models.Teacher); err != nil {
		return nil, err
	}

	updateTeacherInput := &models.TeacherHTTP{
		UserHTTP: userHTTPFromProfileInput(input),
	}
	updateTeacherInput.UserHTTP.Role = 1

	teacherUpdated, updateTeacherErr := r.usersDelegate.UpdateTeacher(updateTeacherInput)
	if updateTeacherErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: updateTeacherErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &teacherUpdated, nil
}

// DeleteTeacher is the resolver for the deleteTeacher field.
func (r *mutationResolver) DeleteTeacher(ctx context.Context, teacherID string) (*models.DeletedTeacher, error) {
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

	deleteTeacherErr := r.usersDelegate.DeleteTeacher(teacherID)
	if deleteTeacherErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteTeacherErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedTeacher{TeacherID: teacherID}, nil
}

// SetTeacherForRobboGroup is the resolver for the SetTeacherForRobboGroup field.
func (r *mutationResolver) SetTeacherForRobboGroup(ctx context.Context, teacherID string, robboGroupID string) (models.TeachersResult, error) {
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

	setTeacherForRobboGroupErr := r.robboGroupDelegate.SetTeacherForRobboGroup(teacherID, robboGroupID)
	if setTeacherForRobboGroupErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: setTeacherForRobboGroupErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	teachers, getTeachersErr := r.usersDelegate.GetTeacherByRobboGroupId(robboGroupID)
	if getTeachersErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeachersErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers:  teachers,
		CountRows: len(teachers),
	}, nil
}

// DeleteTeacherForRobboGroup is the resolver for the DeleteTeacherForRobboGroup field.
func (r *mutationResolver) DeleteTeacherForRobboGroup(ctx context.Context, teacherID string, robboGroupID string) (models.TeachersResult, error) {
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

	deleteTeacherForRobboGroupErr := r.robboGroupDelegate.DeleteTeacherForRobboGroup(teacherID, robboGroupID)
	if deleteTeacherForRobboGroupErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteTeacherForRobboGroupErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	teachers, getTeachersErr := r.usersDelegate.GetTeacherByRobboGroupId(robboGroupID)
	if getTeachersErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeachersErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers:  teachers,
		CountRows: len(teachers),
	}, nil
}

// GetAllTeachers is the resolver for the GetAllTeachers field.
func (r *queryResolver) GetAllTeachers(ctx context.Context, page string, pageSize string) (models.TeachersResult, error) {
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

	teachers, countRows, getAllTeachersErr := r.usersDelegate.GetAllTeachers(page, pageSize)
	if getAllTeachersErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllTeachersErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers:  teachers,
		CountRows: countRows,
	}, nil
}

// GetTeacherByID is the resolver for the GetTeacherById field.
func (r *queryResolver) GetTeacherByID(ctx context.Context, teacherID string) (models.TeacherResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.Teacher, models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	teacher, getTeacherByIdErr := r.usersDelegate.GetTeacherById(teacherID)
	if getTeacherByIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeacherByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return teacher, nil
}

// GetTeachersByStudentID is the resolver for the GetTeachersByStudentId field.
func (r *queryResolver) GetTeachersByStudentID(ctx context.Context, studentID string) (models.TeachersResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.Teacher, models.UnitAdmin, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	teachers, getTeachersByStudentIdErr := r.usersDelegate.GetTeachersByStudentId(studentID)
	if getTeachersByStudentIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeachersByStudentIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers: teachers,
	}, nil
}

// GetTeachersByRobboGroupID is the resolver for the GetTeachersByRobboGroupId field.
func (r *queryResolver) GetTeachersByRobboGroupID(ctx context.Context, robboGroupID string) (models.TeachersResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.SuperAdmin, models.UnitAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	teachers, getTeacherByRobboGroupIdErr := r.usersDelegate.GetTeacherByRobboGroupId(robboGroupID)
	if getTeacherByRobboGroupIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeacherByRobboGroupIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers: teachers,
	}, nil
}

// SearchTeachersByEmail is the resolver for the SearchTeachersByEmail field.
func (r *queryResolver) SearchTeachersByEmail(ctx context.Context, email string, page string, pageSize string) (models.TeachersResult, error) {
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

	teachers, countRows, searchTeachersByEmailErr := r.usersDelegate.SearchTeacherByEmail(email, page, pageSize)
	if searchTeachersByEmailErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: searchTeachersByEmailErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{
		Teachers:  teachers,
		CountRows: countRows,
	}, nil
}
