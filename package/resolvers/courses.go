package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/utils"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// CreateAccessCourseRelationRobboGroup is the resolver for the CreateCourseRelationGroup field.
func (r *mutationResolver) CreateAccessCourseRelationRobboGroup(ctx context.Context, input models.NewAccessCourseRelationRobboGroup) (models.CourseRelationResult, error) {
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

	//_, parseErr := strconv.ParseUint(input.CourseID, 10, 64)
	//if parseErr != nil {
	//	err := courses.ErrIncorrectInputParam
	//	return &models.Error{Message: err.Error()}, err
	//}
	//
	//_, parseErr = strconv.ParseUint(input.RobboGroupID, 10, 64)
	//if parseErr != nil {
	//	err := courses.ErrIncorrectInputParam
	//	return &models.Error{Message: err.Error()}, err
	//}

	courseRelation := &models.CourseRelationHTTP{
		CourseID: input.CourseID,
		ObjectID: input.RobboGroupID,
	}
	newCourseRelation, createAccessCourseRelationRobboGroupErr := r.coursesDelegate.
		CreateAccessCourseRelationRobboGroup(courseRelation)
	if createAccessCourseRelationRobboGroupErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createAccessCourseRelationRobboGroupErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newCourseRelation, nil
}

// CreateAccessCourseRelationRobboUnit is the resolver for the CreateCourseRelationUnit field.
func (r *mutationResolver) CreateAccessCourseRelationRobboUnit(ctx context.Context, input models.NewAccessCourseRelationRobboUnit) (models.CourseRelationResult, error) {
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

	// TODO refactor
	//_, parseErr := strconv.ParseUint(input.CourseID, 10, 64)
	//if parseErr != nil {
	//	err := courses.ErrIncorrectInputParam
	//	return &models.Error{Message: err.Error()}, err
	//}
	//
	//_, parseErr = strconv.ParseUint(input.RobboUnitID, 10, 64)
	//if parseErr != nil {
	//	err := courses.ErrIncorrectInputParam
	//	return &models.Error{Message: err.Error()}, err
	//}

	courseRelation := &models.CourseRelationHTTP{
		CourseID: input.CourseID,
		ObjectID: input.RobboUnitID,
	}
	newCourseRelation, createAccessCourseRelationRobboUnitErr := r.coursesDelegate.
		CreateAccessCourseRelationRobboUnit(courseRelation)
	if createAccessCourseRelationRobboUnitErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createAccessCourseRelationRobboUnitErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newCourseRelation, nil
}

// CreateAccessCourseRelationStudent is the resolver for the CreateCourseRelationStudent field.
func (r *mutationResolver) CreateAccessCourseRelationStudent(ctx context.Context, input models.NewAccessCourseRelationStudent) (models.CourseRelationResult, error) {
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

	courseRelation := &models.CourseRelationHTTP{
		CourseID: input.CourseID,
		ObjectID: input.StudentID,
	}
	newCourseRelation, createAccessCourseRelationStudentErr := r.coursesDelegate.
		CreateAccessCourseRelationStudent(courseRelation)
	if createAccessCourseRelationStudentErr != nil {
		err := createAccessCourseRelationStudentErr
		return &models.Error{Message: err.Error()}, err
	}
	return newCourseRelation, nil
}

// CreateAccessCourseRelationTeacher is the resolver for the CreateCourseRelationTeacher field.
func (r *mutationResolver) CreateAccessCourseRelationTeacher(ctx context.Context, input models.NewAccessCourseRelationTeacher) (models.CourseRelationResult, error) {
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

	courseRelation := &models.CourseRelationHTTP{
		CourseID: input.CourseID,
		ObjectID: input.TeacherID,
	}
	newCourseRelation, createAccessCourseRelationTeacherErr := r.coursesDelegate.
		CreateAccessCourseRelationTeacher(courseRelation)
	if createAccessCourseRelationTeacherErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createAccessCourseRelationTeacherErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newCourseRelation, nil
}

// CreateAccessCourseRelationUnitAdmin is the resolver for the CreateCourseRelationUnitAdmin field.
func (r *mutationResolver) CreateAccessCourseRelationUnitAdmin(ctx context.Context, input models.NewAccessCourseRelationUnitAdmin) (models.CourseRelationResult, error) {
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

	courseRelation := &models.CourseRelationHTTP{
		CourseID: input.CourseID,
		ObjectID: input.UnitAdminID,
	}
	newCourseRelation, createAccessCourseRelationUnitAdminErr := r.coursesDelegate.
		CreateAccessCourseRelationUnitAdmin(courseRelation)
	if createAccessCourseRelationUnitAdminErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: createAccessCourseRelationUnitAdminErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return newCourseRelation, nil
}

// DeleteAccessCourseRelationByID is the resolver for the DeleteCourseRelationById field.
func (r *mutationResolver) DeleteAccessCourseRelationByID(ctx context.Context, courseRelationID string) (*models.DeletedCourseRelation, error) {
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

	_, deleteCourseRelationByIdErr := r.coursesDelegate.DeleteAccessCourseRelationById(courseRelationID)
	if deleteCourseRelationByIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: deleteCourseRelationByIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.DeletedCourseRelation{CourseRelationID: courseRelationID}, nil
}

// GetStudentsAdmittedToTheCourse is the resolver for the GetStudentsAdmittedToTheCourse field.
func (r *queryResolver) GetStudentsAdmittedToTheCourse(ctx context.Context, courseID string, page *string, pageSize *string) (models.StudentsResult, error) {
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

	students, getStudentsAdmittedToTheCourseErr := r.coursesDelegate.
		GetStudentsAdmittedToTheCourse(courseID, page, pageSize)
	if getStudentsAdmittedToTheCourseErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getStudentsAdmittedToTheCourseErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.StudentHTTPList{Students: students}, nil
}

// GetUnitAdminsAdmittedToTheCourse is the resolver for the GetUnitAdminsAdmittedToTheCourse field.
func (r *queryResolver) GetUnitAdminsAdmittedToTheCourse(ctx context.Context, courseID string, page *string, pageSize *string) (models.UnitAdminsResult, error) {
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

	unitAdmins, getUnitAdminsAdmittedToTheCourseErr := r.coursesDelegate.
		GetUnitAdminsAdmittedToTheCourse(courseID, page, pageSize)
	if getUnitAdminsAdmittedToTheCourseErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getUnitAdminsAdmittedToTheCourseErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.UnitAdminHTTPList{UnitAdmins: unitAdmins}, nil
}

// GetTeachersAdmittedToTheCourse is the resolver for the GetTeachersAdmittedToTheCourse field.
func (r *queryResolver) GetTeachersAdmittedToTheCourse(ctx context.Context, courseID string, page *string, pageSize *string) (models.TeachersResult, error) {
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

	teachers, getTeachersAdmittedToTheCourseErr := r.coursesDelegate.
		GetTeachersAdmittedToTheCourse(courseID, page, pageSize)
	if getTeachersAdmittedToTheCourseErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getTeachersAdmittedToTheCourseErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.TeacherHTTPList{Teachers: teachers}, nil
}

// GetRobboGroupsAdmittedToTheCourse is the resolver for the GetRobboGroupsAdmittedToTheCourse field.
func (r *queryResolver) GetRobboGroupsAdmittedToTheCourse(ctx context.Context, courseID string, page *string, pageSize *string) (models.RobboGroupsResult, error) {
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

	robboGroups, getRobboGroupsAdmittedToTheCourseErr := r.coursesDelegate.
		GetRobboGroupsAdmittedToTheCourse(courseID, page, pageSize)
	if getRobboGroupsAdmittedToTheCourseErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getRobboGroupsAdmittedToTheCourseErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.RobboGroupHTTPList{RobboGroups: robboGroups}, nil
}

// GetRobboUnitsAdmittedToTheCourse is the resolver for the GetRobboUnitsAdmittedToTheCourse field.
func (r *queryResolver) GetRobboUnitsAdmittedToTheCourse(ctx context.Context, courseID string, page *string, pageSize *string) (models.RobboUnitsResult, error) {
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
	robboUnits, getRobboUnitsAdmittedToTheCourseErr := r.coursesDelegate.
		GetRobboUnitsAdmittedToTheCourse(courseID, page, pageSize)
	if getRobboUnitsAdmittedToTheCourseErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getRobboUnitsAdmittedToTheCourseErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.RobboUnitHTTPList{RobboUnits: robboUnits}, nil
}

// GetCourseContent is the resolver for the GetCourseContent field.
func (r *queryResolver) GetCourseContent(ctx context.Context, courseID string) (models.CourseResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{
		models.Student,
		models.FreeListener,
		models.Teacher,
		models.UnitAdmin,
		models.SuperAdmin,
	}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	courseHttp, getCourseContentErr := r.coursesDelegate.GetCourseContent(courseID)
	if getCourseContentErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getCourseContentErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return courseHttp, nil
}

// GetCoursesByUser is the resolver for the GetCoursesByUser field.
func (r *queryResolver) GetCoursesByUser(ctx context.Context, page *string, pageSize *string) (models.CoursesResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userId := ginContext.Value("user_id").(string)
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{
		models.Student,
		models.FreeListener,
		models.Teacher,
		models.UnitAdmin,
		models.SuperAdmin,
	}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	courses, getCoursesByUserErr := r.coursesDelegate.
		GetCoursesByUser(userId, userRole, "1", "10")
	if getCoursesByUserErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getCoursesByUserErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return courses, nil
}

// GetCoursesByRobboUnitID is the resolver for the GetCoursesByRobboUnitId field.
func (r *queryResolver) GetCoursesByRobboUnitID(ctx context.Context, robboUnitID string, page *string, pageSize *string) (models.CoursesResult, error) {
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

	courses, getCoursesByUserErr := r.coursesDelegate.
		GetCoursesByRobboUnitId(robboUnitID, utils.UseString(page), utils.UseString(pageSize))
	if getCoursesByUserErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getCoursesByUserErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return courses, nil
}

// GetCoursesByRobboGroupID is the resolver for the GetCoursesByRobboGroupId field.
func (r *queryResolver) GetCoursesByRobboGroupID(ctx context.Context, robboGroupID string, page *string, pageSize *string) (models.CoursesResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{
		models.Teacher,
		models.UnitAdmin,
		models.SuperAdmin,
	}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}

	courses, getCoursesByUserErr := r.coursesDelegate.
		GetCoursesByRobboGroupId(robboGroupID, utils.UseString(page), utils.UseString(pageSize))
	if getCoursesByUserErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getCoursesByUserErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return courses, nil
}

// GetAllPublicCourses is the resolver for the GetAllPublicCourses field.
func (r *queryResolver) GetAllPublicCourses(ctx context.Context, pageNumber string) (models.CoursesResult, error) {
	ginContext, getGinContextErr := GinContextFromContext(ctx)
	if getGinContextErr != nil {
		return nil, getGinContextErr
	}
	userRole := ginContext.Value("user_role").(models.Role)
	allowedRoles := []models.Role{
		models.Student,
		models.Parent,
		models.FreeListener,
		models.Teacher,
		models.UnitAdmin,
		models.SuperAdmin,
	}
	accessErr := r.authDelegate.UserAccess(userRole, allowedRoles, ctx)
	if accessErr != nil {
		return nil, accessErr
	}
	courses, getAllPublicCoursesErr := r.coursesDelegate.GetAllPublicCourses(pageNumber)
	if getAllPublicCoursesErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAllPublicCoursesErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return courses, nil
}

// GetEnrollments is the resolver for the GetEnrollments field.
func (r *queryResolver) GetEnrollments(ctx context.Context, username string) (models.EnrollmentsResult, error) {
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

	enrollments, getEnrollmentsErr := r.coursesDelegate.GetEnrollments(username)
	if getEnrollmentsErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getEnrollmentsErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return enrollments, nil
}

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *queryResolver) GetAccessCourseRelationsByCourseID(ctx context.Context, courseID string) (models.CourseRelationsResult, error) {
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

	courseRelations, getAccessCourseRelationsByCourseIdErr := r.coursesDelegate.GetAccessCourseRelationsByCourseId(courseID)
	if getAccessCourseRelationsByCourseIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAccessCourseRelationsByCourseIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsByRobboUnitID(ctx context.Context, robboUnitID string) (models.CourseRelationsResult, error) {
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

	courseRelations, getAccessCourseRelationsByRobboUnitIdErr := r.coursesDelegate.
		GetAccessCourseRelationsByRobboUnitId(robboUnitID)
	if getAccessCourseRelationsByRobboUnitIdErr != nil {
		err := getAccessCourseRelationsByRobboUnitIdErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsByRobboGroupID(ctx context.Context, robboGroupID string) (models.CourseRelationsResult, error) {
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

	courseRelations, getAccessCourseRelationsByRobboGroupIdErr := r.coursesDelegate.
		GetAccessCourseRelationsByRobboGroupId(robboGroupID)
	if getAccessCourseRelationsByRobboGroupIdErr != nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: getAccessCourseRelationsByRobboGroupIdErr.Error(),
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsByStudentID(ctx context.Context, studentID string) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsByStudentIdErr := r.coursesDelegate.GetAccessCourseRelationsByStudentId(studentID)
	if getAccessCourseRelationsByStudentIdErr != nil {
		err := getAccessCourseRelationsByStudentIdErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsByTeacherID(ctx context.Context, teacherID string) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsByTeacherIdErr := r.coursesDelegate.GetAccessCourseRelationsByTeacherId(teacherID)
	if getAccessCourseRelationsByTeacherIdErr != nil {
		err := getAccessCourseRelationsByTeacherIdErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsByUnitAdminID(ctx context.Context, unitAdminID string) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsByUnitAdminIdErr := r.coursesDelegate.GetAccessCourseRelationsByUnitAdminId(unitAdminID)
	if getAccessCourseRelationsByUnitAdminIdErr != nil {
		err := getAccessCourseRelationsByUnitAdminIdErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsRobboUnits(ctx context.Context) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsRobboUnitsErr := r.coursesDelegate.GetAccessCourseRelationsRobboUnits()
	if getAccessCourseRelationsRobboUnitsErr != nil {
		err := getAccessCourseRelationsRobboUnitsErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsRobboGroups(ctx context.Context) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsRobboGroupsErr := r.coursesDelegate.GetAccessCourseRelationsRobboGroups()
	if getAccessCourseRelationsRobboGroupsErr != nil {
		err := getAccessCourseRelationsRobboGroupsErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsStudents(ctx context.Context) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsStudentsErr := r.coursesDelegate.GetAccessCourseRelationsStudents()
	if getAccessCourseRelationsStudentsErr != nil {
		err := getAccessCourseRelationsStudentsErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsTeachers(ctx context.Context) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsTeachersErr := r.coursesDelegate.GetAccessCourseRelationsTeachers()
	if getAccessCourseRelationsTeachersErr != nil {
		err := getAccessCourseRelationsTeachersErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
func (r *queryResolver) GetAccessCourseRelationsUnitAdmins(ctx context.Context) (models.CourseRelationsResult, error) {
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
	courseRelations, getAccessCourseRelationsUnitAdminsErr := r.coursesDelegate.GetAccessCourseRelationsUnitAdmins()
	if getAccessCourseRelationsUnitAdminsErr != nil {
		err := getAccessCourseRelationsUnitAdminsErr
		return &models.Error{Message: err.Error()}, err
	}
	return &models.CourseRelationHTTPList{
		CourseRelations: courseRelations,
	}, nil
}
