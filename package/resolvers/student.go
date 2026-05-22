package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/utils"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// CreateStudent is the resolver for the createStudent field.
func (r *mutationResolver) CreateStudent(ctx context.Context, input models.NewStudent) (models.StudentResult, error) {
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

	studentInput := models.StudentHTTP{
		UserHTTP: &models.UserHTTP{
			Email:      input.Email,
			Password:   input.Password,
			Firstname:  input.Firstname,
			Lastname:   input.Lastname,
			Middlename: input.Middlename,
			Nickname:   input.Nickname,
			Role:       0,
		},
		RobboUnitID:  utils.UseString(input.RobboUnitID),
		RobboGroupID: utils.UseString(input.RobboGroupID),
	}

	newStudent, createStudentErr := r.usersDelegate.CreateStudent(&studentInput, *input.ParentID)
	if createStudentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createStudentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newStudent, nil
}

// UpdateStudent is the resolver for the updateStudent field.
func (r *mutationResolver) UpdateStudent(ctx context.Context, input models.UpdateProfileInput) (models.StudentResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	accessErr := profileSelfOrAdminAccess(
		ctx, ginContext, r.authDelegate, input.ID, models.Student,
		[]models.Role{models.UnitAdmin, models.SuperAdmin},
	)
	if accessErr != nil {
		return nil, accessErr
	}
	if err := r.validateUsernameImmutable(ctx, input, models.Student); err != nil {
		return nil, err
	}

	updateStudentInput := &models.StudentHTTP{
		UserHTTP: &models.UserHTTP{
			ID:         input.ID,
			Email:      input.Email,
			Firstname:  input.Firstname,
			Lastname:   input.Lastname,
			Middlename: input.Middlename,
			Nickname:   input.Nickname,
			Role:       0,
		},
	}

	studentUpdated, updateStudentErr := r.usersDelegate.UpdateStudent(updateStudentInput)
	if updateStudentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: updateStudentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return studentUpdated, nil
}

// DeleteStudent is the resolver for the deleteStudent field.
func (r *mutationResolver) DeleteStudent(ctx context.Context, studentID string) (*models.DeletedStudent, error) {
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

	deleteStudentErr := r.usersDelegate.DeleteStudent(studentID)
	if deleteStudentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteStudentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedStudent{StudentID: studentID}, nil
}

// SetRobboGroupIDForStudent is the resolver for the SetRobboGroupIdForStudent field.
func (r *mutationResolver) SetRobboGroupIDForStudent(ctx context.Context, studentID string, robboGroupID string, robboUnitID string) (*models.Error, error) {
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

	addStudentToRobboGroupErr := r.usersDelegate.AddStudentToRobboGroup(studentID, robboGroupID, robboUnitID)
	if addStudentToRobboGroupErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: addStudentToRobboGroupErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return nil, nil
}

// CreateStudentTeacherRelation is the resolver for the CreateStudentTeacherRelation field.
func (r *mutationResolver) CreateStudentTeacherRelation(ctx context.Context, studentID string, teacherID string) (models.StudentResult, error) {
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

	student, createStudentTeacherRelationErr := r.usersDelegate.CreateStudentTeacherRelation(studentID, teacherID)
	if createStudentTeacherRelationErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createStudentTeacherRelationErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return student, nil
}

// DeleteStudentTeacherRelation is the resolver for the DeleteStudentTeacherRelation field.
func (r *mutationResolver) DeleteStudentTeacherRelation(ctx context.Context, studentID string, teacherID string) (models.StudentResult, error) {
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

	student, createStudentTeacherRelationErr := r.usersDelegate.DeleteStudentTeacherRelation(studentID, teacherID)
	if createStudentTeacherRelationErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createStudentTeacherRelationErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return student, nil
}

// GetStudentsByParentID is the resolver for the GetStudentsByParentId field.
func (r *queryResolver) GetStudentsByParentID(ctx context.Context, parentID string) (models.StudentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin, models.Parent}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	students, getStudentByParentIdErr := r.usersDelegate.GetStudentByParentId(parentID)
	if getStudentByParentIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentByParentIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students: students,
	}, nil
}

// GetStudentByID is the resolver for the GetStudentById field.
func (r *queryResolver) GetStudentByID(ctx context.Context, studentID string) (models.StudentResult, error) {
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

	student, getStudentByIdErr := r.usersDelegate.GetStudentById(studentID)
	if getStudentByIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return student, nil
}

// GetStudentsByRobboGroup is the resolver for the GetStudentsByRobboGroup field.
func (r *queryResolver) GetStudentsByRobboGroup(ctx context.Context, robboGroupID string) (models.StudentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.UnitAdmin, models.SuperAdmin, models.Teacher}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	students, getStudentByRobboGroupErr := r.usersDelegate.GetStudentsByRobboGroupId(robboGroupID)
	if getStudentByRobboGroupErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentByRobboGroupErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students: students,
	}, nil
}

// GetStudentsByRobboUnitID is the resolver for the GetStudentsByRobboUnitId field.
func (r *queryResolver) GetStudentsByRobboUnitID(ctx context.Context, robboUnitID string) (models.StudentsResult, error) {
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

	students, getStudentsByRobboUnitIdErr := r.usersDelegate.GetStudentsByRobboUnitId(robboUnitID)
	if getStudentsByRobboUnitIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentsByRobboUnitIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students: students,
	}, nil
}

// GetStudentsByTeacherID is the resolver for the GetStudentsByTeacherId field.
func (r *queryResolver) GetStudentsByTeacherID(ctx context.Context, teacherID string) (models.StudentsResult, error) {
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

	students, getStudentsByTeacherIdErr := r.usersDelegate.GetStudentsByTeacherId(teacherID)
	if getStudentsByTeacherIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentsByTeacherIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students: students,
	}, nil
}

// GetIndividualStudentsByTeacherID is the resolver for the GetIndividualStudentsByTeacherId field.
func (r *queryResolver) GetIndividualStudentsByTeacherID(ctx context.Context, teacherID string) (models.StudentsResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{models.Teacher, models.SuperAdmin}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	students, getIndividualStudentsByTeacherIdErr := r.usersDelegate.GetIndividualStudentsByTeacherId(teacherID)
	if getIndividualStudentsByTeacherIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getIndividualStudentsByTeacherIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students: students,
	}, nil
}

// SearchStudentsByEmail is the resolver for the SearchStudentsByEmail field.
func (r *queryResolver) SearchStudentsByEmail(ctx context.Context, email string, page string, pageSize string) (models.StudentsResult, error) {
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

	students, countRows, searchStudentByEmailErr := r.usersDelegate.SearchStudentByEmail(email, page, pageSize)
	if searchStudentByEmailErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: searchStudentByEmailErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{
		Students:  students,
		CountRows: countRows,
	}, nil
}
