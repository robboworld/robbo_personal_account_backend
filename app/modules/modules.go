package modules

import (
	"context"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/adminaudit"
	adminauditgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/adminaudit/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	acgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent/gateway"
	achttp "github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent/http"
	acusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent/usecase"
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
	"github.com/skinnykaen/robbo_student_personal_account.git/package/dbmigrate"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	edxusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/edx/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	licdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/delegate"
	licgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/gateway"
	lichttp "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/http"
	licusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	moddelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/moderation/delegate"
	modgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/moderation/gateway"
	modhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/moderation/http"
	modusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/moderation/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	notificationgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/notifications/gateway"
	notificationhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/notifications/http"
	notificationusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/notifications/usecase"
	oidchttp "github.com/skinnykaen/robbo_student_personal_account.git/package/oidc/http"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
	paydelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/payments/delegate"
	paygateway "github.com/skinnykaen/robbo_student_personal_account.git/package/payments/gateway"
	payhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/payments/http"
	payusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/payments/usecase"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	ppageaccess "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/access"
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
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/streak"
	streakgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/streak/gateway"
	streakusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/streak/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	tcgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass/gateway"
	tchtpp "github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass/http"
	tcusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/blobstore"
	ufgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/gateway"
	ufhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/http"
	ufusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	usersdelegate "github.com/skinnykaen/robbo_student_personal_account.git/package/users/delegate"
	usersgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/users/gateway"
	usershtpp "github.com/skinnykaen/robbo_student_personal_account.git/package/users/http"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users/profileaccess"
	usersusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/users/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
	usersearchhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch/http"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type GatewayModule struct {
	AuthGateway          auth.Gateway
	CohortsGateway       cohorts.Gateway
	CoursePacketGateway  coursePacket.Gateway
	CoursesGateway       courses.Gateway
	ProjectPageGateway   projectPage.Gateway
	NotificationsGateway notifications.Gateway
	ProjectsGateway      projects.Gateway
	LicensingGateway     licensing.Gateway
	ModerationGateway    moderation.Gateway
	PaymentsGateway      payments.Gateway
	StreakGateway        streak.Gateway
	TeacherClassGateway  teacherclass.Gateway
	RobboGroupGateway    robboGroup.Gateway
	RobboUnitsGateway    robboUnits.Gateway
	UsersGateway         users.Gateway
	UserFilesGateway     userfiles.Gateway
	AdminAuditGateway    adminaudit.Gateway
	AdminContentGateway  admincontent.Gateway
}

func SetupGateway(postgresClient db_client.PostgresClient) (GatewayModule, error) {
	// Bring the projects and licensing schemas up to the code before any gateway uses them.
	if err := dbmigrate.ApplyConfigured(); err != nil {
		return GatewayModule{}, err
	}
	tcGw, err := tcgateway.SetupTeacherClassGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	ppageGw, err := ppagegateway.SetupProjectPageGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	notificationGw, err := notificationgateway.SetupNotificationGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	projectsGw, err := prjgateway.SetupProjectsGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	licensingGw, err := licgateway.SetupLicensingGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	moderationGw, err := modgateway.SetupBansGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	paymentsGw, err := paygateway.SetupPaymentsGateway(postgresClient)
	if err != nil {
		return GatewayModule{}, err
	}
	userFilesGw, err := ufgateway.SetupUserFilesGateway()
	if err != nil {
		return GatewayModule{}, err
	}
	adminAuditGw, err := adminauditgateway.SetupAdminAuditGateway()
	if err != nil {
		return GatewayModule{}, err
	}
	adminContentGw, err := acgateway.SetupAdminContentGateway()
	if err != nil {
		return GatewayModule{}, err
	}
	// A file handed in for homework or given to a class as a starter file stays until the
	// class no longer needs it.
	userfiles.FileInUse = tcGw.Gateway.FileInUse
	ppageaccess.CohortTeacherReader = func(teacherID, courseID, cohortID string) bool {
		inv, err := tcGw.Gateway.GetInviteByCourseCohort(courseID, cohortID)
		if err != nil || inv == nil || inv.ArchivedAt != nil {
			return false
		}
		if inv.OwnerTeacherID == teacherID {
			return true
		}
		coTeacher, err := tcGw.Gateway.IsCoTeacher(inv.ID, teacherID)
		return err == nil && coTeacher
	}
	profileaccess.TeacherStudentReader = func(teacherID, studentID string) bool {
		studentNum, err := strconv.ParseInt(strings.TrimSpace(studentID), 10, 64)
		if err != nil || studentNum <= 0 {
			return false
		}
		invites, err := tcGw.Gateway.ListInvitesByTeacher(teacherID)
		if err != nil {
			return false
		}
		var cohortIDs []int64
		var localIDs []string
		for _, inv := range invites {
			if inv.ArchivedAt != nil {
				continue
			}
			if !inv.LmsLinked {
				localIDs = append(localIDs, inv.ID)
				continue
			}
			if id, err := strconv.ParseInt(strings.TrimSpace(inv.EdxCohortID), 10, 64); err == nil {
				cohortIDs = append(cohortIDs, id)
			}
		}
		if member, err := tcGw.Gateway.IsMemberOfAny(strconv.FormatInt(studentNum, 10), localIDs); err == nil && member {
			return true
		}
		if len(cohortIDs) == 0 {
			return false
		}
		reader, err := lmsdb.NewReaderFromConfig()
		if err != nil {
			return false
		}
		defer reader.Close()
		member, err := reader.IsUserInAnyCohort(studentNum, cohortIDs)
		return err == nil && member
	}
	return GatewayModule{
		AuthGateway:          authgateway.SetupAuthGateway(postgresClient),
		CohortsGateway:       chrtgateway.SetupCohortsGateway(postgresClient),
		CoursePacketGateway:  coursePacketgateway.SetupCoursePacketGateway(postgresClient),
		CoursesGateway:       crsgateway.SetupCoursesGateway(postgresClient),
		ProjectPageGateway:   ppageGw,
		NotificationsGateway: notificationGw,
		ProjectsGateway:      projectsGw,
		LicensingGateway:     licensingGw.Gateway,
		ModerationGateway:    moderationGw,
		PaymentsGateway:      paymentsGw,
		StreakGateway:        streakgateway.SetupStreakGateway(postgresClient).Gateway,
		TeacherClassGateway:  tcGw.Gateway,
		RobboGroupGateway:    robboGroupgateway.SetupRobboGroupGateway(postgresClient),
		RobboUnitsGateway:    robboUnitsgateway.SetupRobboUnitsGateway(postgresClient),
		UsersGateway:         usersgateway.SetupUsersGateway(postgresClient),
		UserFilesGateway:     userFilesGw,
		AdminAuditGateway:    adminAuditGw,
		AdminContentGateway:  adminContentGw,
	}, nil
}

type UseCaseModule struct {
	AuthUseCase          auth.UseCase
	CohortsUseCase       cohorts.UseCase
	CoursePacketUseCase  coursePacket.UseCase
	CoursesUseCase       courses.UseCase
	EdxUseCase           edx.UseCase
	ProjectPageUseCase   projectPage.UseCase
	NotificationsUseCase notifications.UseCase
	ProjectsUseCase      projects.UseCase
	LicensingUseCase     licensing.UseCase
	ModerationUseCase    moderation.UseCase
	PaymentsUseCase      payments.UseCase
	StreakUseCase        streak.UseCase
	TeacherClassUseCase  teacherclass.UseCase
	RobboGroupUseCase    robboGroup.UseCase
	RobboUnitsUseCase    robboUnits.UseCase
	UsersUseCase         users.UseCase
	UserFilesUseCase     userfiles.UseCase
	// StorageQuota is the one account limit over projects and files.
	StorageQuota        *storage.Quota
	AdminContentUseCase *acusecase.AdminContentUseCase
	StorageAlerter      *acusecase.StorageAlerter
}

// setupUserFiles opens the blob store only when the feature is on, so a stand without the
// volume still starts; when on, a missing or read-only directory stops the start.
// The volume is nil while the feature is off.
func setupUserFiles(gateway GatewayModule, quota *storage.Quota) (userfiles.UseCase, admincontent.Volume, error) {
	cfg := ufusecase.ConfigFromViper()
	var blobs blobstore.BlobStore
	var volume admincontent.Volume
	if cfg.Enabled {
		fs, err := blobstore.NewLocalFS(viper.GetString("userFiles.storageDir"))
		if err != nil {
			return nil, nil, err
		}
		blobs, volume = fs, fs
	}
	return ufusecase.NewUserFilesUseCase(cfg, gateway.UserFilesGateway, blobs, quota, gateway.LicensingGateway), volume, nil
}

func SetupUseCase(gateway GatewayModule, portalGateway portalgateway.Gateway, userSearch *usersearch.Service) (UseCaseModule, error) {
	licensingUC, err := licusecase.SetupLicensingUseCase(gateway.LicensingGateway)
	if err != nil {
		return UseCaseModule{}, err
	}
	streakUC := streakusecase.SetupStreakUseCase(gateway.StreakGateway)
	edxMod := edxusecase.SetupEdxApiUseCase()
	quota := storage.NewQuota(gateway.LicensingGateway, gateway.ProjectPageGateway, gateway.UserFilesGateway)
	userFilesUC, volume, err := setupUserFiles(gateway, quota)
	if err != nil {
		return UseCaseModule{}, err
	}
	tcUC := tcusecase.SetupTeacherClassUseCase(gateway.TeacherClassGateway, edxMod.UseCase, userFilesUC, gateway.NotificationsGateway)
	notifications.ClassMessageRecorder = tcUC.UseCase.RecordClassMessage
	projectPage.TeacherEditRecorder = tcUC.UseCase.RecordTeacherEdit
	projectPageUC := ppageusecase.SetupProjectPageUseCase(
		gateway.ProjectPageGateway,
		gateway.ProjectsGateway,
		gateway.NotificationsGateway,
		gateway.LicensingGateway,
		quota,
		gateway.AdminAuditGateway,
		gateway.UsersGateway,
	).UseCase
	uploadsStop := ufusecase.ConfigFromViper().MinFreeDiskMB << 20
	adminContentUC := acusecase.NewAdminContentUseCase(acusecase.Deps{
		Gateway:               gateway.AdminContentGateway,
		Audit:                 gateway.AdminAuditGateway,
		Projects:              gateway.ProjectPageGateway,
		ProjectsUC:            projectPageUC,
		Files:                 userFilesUC,
		Quota:                 quota,
		Licensing:             gateway.LicensingGateway,
		Notifications:         gateway.NotificationsGateway,
		Volume:                volume,
		UploadsStopBelowBytes: uploadsStop,
	})
	var alerter *acusecase.StorageAlerter
	if volume != nil {
		alerter = acusecase.NewStorageAlerter(volume, gateway.NotificationsGateway, acusecase.LMSSuperAdmins, uploadsStop)
	}
	return UseCaseModule{
		AuthUseCase: authusecase.SetupAuthUseCase(
			gateway.UsersGateway, portalGateway, gateway.LicensingGateway, streakUC.UseCase,
		),
		CohortsUseCase:      chrtusecase.SetupCohortUseCase(gateway.CohortsGateway),
		CoursePacketUseCase: coursePacketusecase.SetupCoursePacketUseCase(gateway.CoursePacketGateway),
		CoursesUseCase: crsusecase.SetupCourseUseCase(
			gateway.CoursesGateway,
			gateway.UsersGateway,
			gateway.RobboUnitsGateway,
			gateway.RobboGroupGateway,
		),
		EdxUseCase: edxMod.UseCase,
		NotificationsUseCase: notificationusecase.SetupNotificationUseCase(
			gateway.NotificationsGateway, notificationgateway.NewLMSGatewayFromConfig(),
		),
		ProjectPageUseCase:  projectPageUC,
		ProjectsUseCase:     prjusecase.SetupProjectUseCase(gateway.ProjectsGateway),
		LicensingUseCase:    licensingUC.UseCase,
		ModerationUseCase:   modusecase.SetupBanUseCase(gateway.ModerationGateway, gateway.LicensingGateway, userSearch).UseCase,
		PaymentsUseCase:     payusecase.SetupPaymentsUseCase(gateway.PaymentsGateway, licensingUC.UseCase).UseCase,
		StreakUseCase:       streakUC.UseCase,
		TeacherClassUseCase: tcUC.UseCase,
		RobboGroupUseCase:   robboGroupusecase.SetupRobboGroupUseCase(gateway.RobboGroupGateway, gateway.UsersGateway),
		RobboUnitsUseCase:   robboUnitsusecase.SetupRobboUnitsUseCase(gateway.RobboUnitsGateway, gateway.UsersGateway),
		UsersUseCase:        usersusecase.SetupUsersUseCase(gateway.UsersGateway, gateway.RobboGroupGateway),
		UserFilesUseCase:    userFilesUC,
		StorageQuota:        quota,
		AdminContentUseCase: adminContentUC,
		StorageAlerter:      alerter,
	}, nil
}

type DelegateModule struct {
	AuthDelegate         auth.Delegate
	CohortsDelegate      cohorts.Delegate
	CoursePacketDelegate coursePacket.Delegate
	CoursesDelegate      courses.Delegate
	ProjectPageDelegate  projectPage.Delegate
	ProjectsDelegate     projects.Delegate
	LicensingDelegate    licensing.Delegate
	ModerationDelegate   moderation.Delegate
	PaymentsDelegate     payments.Delegate
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
		LicensingDelegate:    licdelegate.SetupLicensingDelegate(usecase.LicensingUseCase),
		ModerationDelegate:   moddelegate.SetupModerationDelegate(usecase.ModerationUseCase),
		PaymentsDelegate:     paydelegate.SetupPaymentsDelegate(usecase.PaymentsUseCase),
		RobboGroupDelegate:   robboGroupdelegate.SetupRobboGroupDelegate(usecase.RobboGroupUseCase),
		RobboUnitsDelegate:   robboUnitsdelegate.SetupRobboUnitsDelegate(usecase.RobboUnitsUseCase),
		UsersDelegate:        usersdelegate.SetupUsersDelegate(usecase.UsersUseCase),
	}
}

type HandlerModule struct {
	ProjectsHandler      prjhttp.Handler
	ProjectPageHandler   ppagehttp.Handler
	AuthHandler          authhttp.Handler
	CoursesHandler       crshttp.Handler
	CohortsHandler       chrthttp.Handler
	UsersHandler         usershtpp.Handler
	RobboUnitsHandler    robboUnitshttp.Handler
	RobboGroupHandler    robboGrouphttp.Handler
	CoursePacketHandler  coursePackethttp.Handler
	LicensingHandler     lichttp.Handler
	PaymentsHandler      payhttp.Handler
	NotificationsHandler notificationhttp.Handler
	UserSearchHandler    usersearchhttp.Handler
	ModerationHandler    modhttp.Handler
	TeacherClassHandler  tchtpp.Handler
	UserFilesHandler     ufhttp.Handler
	AdminContentHandler  achttp.Handler
	OIDCHandler          *oidchttp.Handler
	LicensingGateway     licensing.Gateway
}

func SetupUserSearchService() *usersearch.Service {
	return usersearch.NewFromConfig()
}

func StartUserSearchSync(service *usersearch.Service, lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if service != nil {
				service.Stop()
			}
			return nil
		},
	})
}

func StartBanExpiryWorker(usecase UseCaseModule, lc fx.Lifecycle) {
	modusecase.StartBanExpiryWorker(usecase.ModerationUseCase, lc)
}

func StartSb3RetentionWorker(usecase UseCaseModule, lc fx.Lifecycle) {
	ppageusecase.StartSb3RetentionWorker(usecase.ProjectPageUseCase, lc)
}

func StartUserFilesGCWorker(usecase UseCaseModule, lc fx.Lifecycle) {
	ufusecase.StartGCWorker(usecase.UserFilesUseCase, lc)
}

func StartCoTeacherReminderWorker(usecase UseCaseModule, lc fx.Lifecycle) {
	tcusecase.StartCoTeacherReminderWorker(usecase.TeacherClassUseCase, lc)
}

func StartStorageAlertWorker(usecase UseCaseModule, lc fx.Lifecycle) {
	acusecase.StartStorageAlertWorker(usecase.StorageAlerter, lc)
}

func SetupHandler(
	delegate DelegateModule,
	usecase UseCaseModule,
	gateway GatewayModule,
	oidcHandler *oidchttp.Handler,
	userSearch *usersearch.Service,
) HandlerModule {
	return HandlerModule{
		ProjectsHandler: prjhttp.NewProjectsHandler(delegate.AuthDelegate, delegate.ProjectsDelegate, delegate.ProjectPageDelegate),
		ProjectPageHandler: ppagehttp.NewProjectPageHandler(
			delegate.AuthDelegate,
			delegate.ProjectsDelegate,
			delegate.ProjectPageDelegate,
		),
		AuthHandler:         authhttp.NewAuthHandler(delegate.AuthDelegate, usecase.StreakUseCase),
		CoursesHandler:      crshttp.NewCoursesHandler(delegate.AuthDelegate, delegate.CoursesDelegate),
		CohortsHandler:      chrthttp.NewCohortsHandler(delegate.AuthDelegate, delegate.CohortsDelegate),
		UsersHandler:        usershtpp.NewUsersHandler(delegate.AuthDelegate, delegate.UsersDelegate),
		RobboUnitsHandler:   robboUnitshttp.NewRobboUnitsHandler(delegate.AuthDelegate, delegate.RobboUnitsDelegate),
		RobboGroupHandler:   robboGrouphttp.NewRobboGroupHandler(delegate.AuthDelegate, delegate.RobboGroupDelegate),
		CoursePacketHandler: coursePackethttp.NewCoursePacketHandler(delegate.AuthDelegate, delegate.CoursePacketDelegate),
		LicensingHandler: lichttp.NewLicensingHandler(
			delegate.AuthDelegate, delegate.LicensingDelegate, gateway.ProjectPageGateway,
			usecase.StorageQuota, usecase.UserFilesUseCase,
		),
		PaymentsHandler:      payhttp.NewPaymentsHandler(delegate.AuthDelegate, delegate.PaymentsDelegate),
		NotificationsHandler: notificationhttp.NewNotificationHandler(delegate.AuthDelegate, usecase.NotificationsUseCase),
		UserSearchHandler:    usersearchhttp.NewHandler(delegate.AuthDelegate, userSearch),
		ModerationHandler:    modhttp.NewHandler(delegate.AuthDelegate, delegate.ModerationDelegate),
		TeacherClassHandler:  tchtpp.NewTeacherClassHandler(delegate.AuthDelegate, usecase.TeacherClassUseCase),
		UserFilesHandler:     ufhttp.NewUserFilesHandler(delegate.AuthDelegate, usecase.UserFilesUseCase),
		AdminContentHandler:  achttp.NewAdminContentHandler(delegate.AuthDelegate, usecase.AdminContentUseCase),
		OIDCHandler:          oidcHandler,
		LicensingGateway:     gateway.LicensingGateway,
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
