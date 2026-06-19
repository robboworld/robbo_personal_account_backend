package modules

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	authdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/auth/delegate"
	authgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/auth/gateway"
	authhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/auth/http"
	authusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/auth/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/cohorts"
	chrtdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/cohorts/delegate"
	chrtgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/cohorts/gateway"
	chrthttp "github.com/skinnykaen/robbo_student_personal_account.git/package/cohorts/http"
	chrtusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/cohorts/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket"
	coursePacketdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket/delegate"
	coursePacketgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket/gateway"
	coursePackethttp "github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket/http"
	coursePacketusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/coursePacket/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/courses"
	crsdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/courses/delegate"
	crsgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/courses/gateway"
	crshttp "github.com/skinnykaen/robbo_student_personal_account.git/package/courses/http"
	crsusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/courses/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	edxusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/edx/usecase"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	oidchttp "github.com/skinnykaen/robbo_student_personal_account.git/package/oidc/http"
	portalhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/http"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	ppagedelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/delegate"
	ppagegateway "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/gateway"
	ppagehttp "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/http"
	ppageusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projects"
	prjdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/projects/delegate"
	prjgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/projects/gateway"
	prjhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/projects/http"
	prjusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/projects/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/resolvers"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup"
	robboGroupdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup/delegate"
	robboGroupgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup/gateway"
	robboGrouphttp "github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup/http"
	robboGroupusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits"
	robboUnitsdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits/delegate"
	robboUnitsgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits/gateway"
	robboUnitshttp "github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits/http"
	robboUnitsusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	usersdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/users/delegate"
	usersgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/users/gateway"
	usershtpp "github.com/skinnykaen/robbo_student_personal_account.git/package/users/http"
	usersusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/users/usecase"
)

type GatewayModule struct {
	AuthGateway         auth.Gateway
	CohortsGateway      cohorts.Gateway
	CoursePacketGateway coursePacket.Gateway
	CoursesGateway      courses.Gateway
	ProjectPageGateway  projectPage.Gateway
	ProjectsGateway     projects.Gateway
	RobboGroupGateway   robboGroup.Gateway
	RobboUnitsGateway   robboUnits.Gateway
	UsersGateway        users.Gateway
}

func SetupGateway(postgresClient db_client.PostgresClient) GatewayModule {
	return GatewayModule{
		AuthGateway:         authgateway.SetupAuthGateway(postgresClient),
		CohortsGateway:      chrtgateway.SetupCohortsGateway(postgresClient),
		CoursePacketGateway: coursePacketgateway.SetupCoursePacketGateway(postgresClient),
		CoursesGateway:      crsgateway.SetupCoursesGateway(postgresClient),
		ProjectPageGateway:  ppagegateway.SetupProjectPageGateway(postgresClient),
		ProjectsGateway:     prjgateway.SetupProjectsGateway(postgresClient),
		RobboGroupGateway:   robboGroupgateway.SetupRobboGroupGateway(postgresClient),
		RobboUnitsGateway:   robboUnitsgateway.SetupRobboUnitsGateway(postgresClient),
		UsersGateway:        usersgateway.SetupUsersGateway(postgresClient),
	}
}

type UseCaseModule struct {
	AuthUseCase         auth.UseCase
	CohortsUseCase      cohorts.UseCase
	CoursePacketUseCase coursePacket.UseCase
	CoursesUseCase      courses.UseCase
	EdxUseCase          edx.UseCase
	ProjectPageUseCase  projectPage.UseCase
	ProjectsUseCase     projects.UseCase
	RobboGroupUseCase   robboGroup.UseCase
	RobboUnitsUseCase   robboUnits.UseCase
	UsersUseCase        users.UseCase
}

func SetupUseCase(gateway GatewayModule, portalGateway portalgateway.Gateway) UseCaseModule {
	return UseCaseModule{
		AuthUseCase:         authusecase.SetupAuthUseCase(gateway.UsersGateway, portalGateway),
		CohortsUseCase:      chrtusecase.SetupCohortUseCase(gateway.CohortsGateway),
		CoursePacketUseCase: coursePacketusecase.SetupCoursePacketUseCase(gateway.CoursePacketGateway),
		CoursesUseCase: crsusecase.SetupCourseUseCase(
			gateway.CoursesGateway,
			gateway.UsersGateway,
			gateway.RobboUnitsGateway,
			gateway.RobboGroupGateway,
		),
		EdxUseCase:         edxusecase.SetupEdxApiUseCase(),
		ProjectPageUseCase: ppageusecase.SetupProjectPageUseCase(gateway.ProjectPageGateway, gateway.ProjectsGateway),
		ProjectsUseCase:    prjusecase.SetupProjectUseCase(gateway.ProjectsGateway),
		RobboGroupUseCase:  robboGroupusecase.SetupRobboGroupUseCase(gateway.RobboGroupGateway, gateway.UsersGateway),
		RobboUnitsUseCase:  robboUnitsusecase.SetupRobboUnitsUseCase(gateway.RobboUnitsGateway, gateway.UsersGateway),
		UsersUseCase:       usersusecase.SetupUsersUseCase(gateway.UsersGateway, gateway.RobboGroupGateway),
	}
}

type DelegateModule struct {
	AuthDelegate         auth.Delegate
	CohortsDelegate      cohorts.Delegate
	CoursePacketDelegate coursePacket.Delegate
	CoursesDelegate      courses.Delegate
	ProjectPageDelegate  projectPage.Delegate
	ProjectsDelegate     projects.Delegate
	RobboGroupDelegate   robboGroup.Delegate
	RobboUnitsDelegate   robboUnits.Delegate
	UsersDelegate        users.Delegate
}

func SetupDelegate(usecase UseCaseModule) DelegateModule {
	return DelegateModule{
		AuthDelegate:         authdelegate.SetupAuthDelegate(usecase.AuthUseCase),
		CohortsDelegate:      chrtdelegate.SetupCohortDelegate(usecase.CohortsUseCase, usecase.EdxUseCase),
		CoursePacketDelegate: coursePacketdelegate.SetupCoursePacketDelegate(usecase.CoursePacketUseCase),
		CoursesDelegate:      crsdelegate.SetupCourseDelegate(usecase.CoursesUseCase, usecase.EdxUseCase),
		ProjectPageDelegate:  ppagedelegate.SetupProjectPageDelegate(usecase.ProjectPageUseCase),
		ProjectsDelegate:     prjdelegate.SetupProjectDelegate(usecase.ProjectsUseCase),
		RobboGroupDelegate:   robboGroupdelegate.SetupRobboGroupDelegate(usecase.RobboGroupUseCase),
		RobboUnitsDelegate:   robboUnitsdelegate.SetupRobboUnitsDelegate(usecase.RobboUnitsUseCase),
		UsersDelegate:        usersdelegate.SetupUsersDelegate(usecase.UsersUseCase),
	}
}

type HandlerModule struct {
	ProjectsHandler              prjhttp.Handler
	ProjectPageHandler           ppagehttp.Handler
	AuthHandler                  authhttp.Handler
	CoursesHandler               crshttp.Handler
	CohortsHandler               chrthttp.Handler
	UsersHandler                 usershtpp.Handler
	RobboUnitsHandler            robboUnitshttp.Handler
	RobboGroupHandler            robboGrouphttp.Handler
	CoursePacketHandler          coursePackethttp.Handler
	PortalNotificationsHandler   portalhttp.NotificationsHandler
	OIDCHandler                  *oidchttp.Handler
}

func SetupHandler(
	delegate DelegateModule,
	portalNotifications portalhttp.NotificationsHandler,
	oidcHandler *oidchttp.Handler,
) HandlerModule {
	return HandlerModule{
		ProjectsHandler: prjhttp.NewProjectsHandler(delegate.AuthDelegate, delegate.ProjectsDelegate, delegate.ProjectPageDelegate),
		ProjectPageHandler: ppagehttp.NewProjectPageHandler(
			delegate.AuthDelegate,
			delegate.ProjectsDelegate,
			delegate.ProjectPageDelegate,
		),
		AuthHandler:         authhttp.NewAuthHandler(delegate.AuthDelegate),
		CoursesHandler:      crshttp.NewCoursesHandler(delegate.AuthDelegate, delegate.CoursesDelegate),
		CohortsHandler:      chrthttp.NewCohortsHandler(delegate.AuthDelegate, delegate.CohortsDelegate),
		UsersHandler:        usershtpp.NewUsersHandler(delegate.AuthDelegate, delegate.UsersDelegate),
		RobboUnitsHandler:   robboUnitshttp.NewRobboUnitsHandler(delegate.AuthDelegate, delegate.RobboUnitsDelegate),
		RobboGroupHandler:   robboGrouphttp.NewRobboGroupHandler(delegate.AuthDelegate, delegate.RobboGroupDelegate),
		CoursePacketHandler:        coursePackethttp.NewCoursePacketHandler(delegate.AuthDelegate, delegate.CoursePacketDelegate),
		PortalNotificationsHandler: portalNotifications,
		OIDCHandler:                oidcHandler,
	}
}

type GraphQLModule struct {
	UsersResolver resolvers.Resolver
}

func SetupGraphQLModule(delegate DelegateModule) GraphQLModule {
	return GraphQLModule{
		UsersResolver: resolvers.NewResolver(
			delegate.AuthDelegate,
			delegate.UsersDelegate,
			delegate.RobboGroupDelegate,
			delegate.RobboUnitsDelegate,
			delegate.CoursesDelegate,
			delegate.ProjectPageDelegate,
		),
	}
}
