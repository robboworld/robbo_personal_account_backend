package usecase

import (
	"errors"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/courses"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	"go.uber.org/fx"
	"log"
)

type CourseUseCaseImpl struct {
	courseGateway     courses.Gateway
	robboUnitGateway  robboUnits.Gateway
	robboGroupGateway robboGroup.Gateway
	usersGateway      users.Gateway
}

func (p *CourseUseCaseImpl) GetUnitAdminsAdmittedToTheCourse(courseId string, page *string, pageSize *string) (
	unitAdmins []*models.UnitAdminCore,
	err error,
) {
	courseAccessRelations, getRelationsErr := p.courseGateway.GetAccessCourseRelations(courseId, "unit_admin")
	if getRelationsErr != nil {
		err = getRelationsErr
		return
	}

	for _, courseAccessRelation := range courseAccessRelations {
		unitAdmin, getUnitAdminErr := p.usersGateway.GetUnitAdminById(courseAccessRelation.ObjectId)
		if getUnitAdminErr != nil {
			err = getUnitAdminErr
			return
		}
		unitAdmins = append(unitAdmins, unitAdmin)
	}
	return
}

func (p *CourseUseCaseImpl) GetTeachersAdmittedToTheCourse(courseId string, page *string, pageSize *string) (
	teachers []*models.TeacherCore,
	err error,
) {
	courseAccessRelations, getRelationsErr := p.courseGateway.GetAccessCourseRelations(courseId, "teacher")
	if getRelationsErr != nil {
		err = getRelationsErr
		return
	}

	for _, courseAccessRelation := range courseAccessRelations {
		teacher, getTeacherErr := p.usersGateway.GetTeacherById(courseAccessRelation.ObjectId)
		if getTeacherErr != nil {
			err = getTeacherErr
			return
		}
		teachers = append(teachers, &teacher)
	}
	return
}

func (p *CourseUseCaseImpl) GetRobboUnitsAdmittedToTheCourse(courseId string, page *string, pageSize *string) (
	robboUnits []*models.RobboUnitCore,
	err error,
) {
	courseAccessRelations, getRelationsErr := p.courseGateway.GetAccessCourseRelations(courseId, "robbo_unit")
	if getRelationsErr != nil {
		err = getRelationsErr
		return
	}

	for _, courseAccessRelation := range courseAccessRelations {
		robboUnit, getRobboUnitErr := p.robboUnitGateway.GetRobboUnitById(courseAccessRelation.ObjectId)
		if getRobboUnitErr != nil {
			err = getRobboUnitErr
			return
		}
		robboUnits = append(robboUnits, robboUnit)
	}
	return
}

func (p *CourseUseCaseImpl) GetRobboGroupsAdmittedToTheCourse(courseId string, page *string, pageSize *string) (
	robboGroups []*models.RobboGroupCore,
	err error,
) {
	courseAccessRelations, getRelationsErr := p.courseGateway.GetAccessCourseRelations(courseId, "robbo_group")
	if getRelationsErr != nil {
		err = getRelationsErr
		return
	}

	for _, courseAccessRelation := range courseAccessRelations {
		robboGroup, getRobboGroupErr := p.robboGroupGateway.GetRobboGroupById(courseAccessRelation.ObjectId)
		if getRobboGroupErr != nil {
			err = getRobboGroupErr
			return
		}
		robboGroups = append(robboGroups, robboGroup)
	}
	return
}

func (p *CourseUseCaseImpl) GetStudentsAdmittedToTheCourse(courseId string, page *string, pageSize *string) (
	students []*models.StudentCore, err error,
) {
	courseAccessRelations, getRelationsErr := p.courseGateway.GetAccessCourseRelations(courseId, "student")
	if getRelationsErr != nil {
		err = getRelationsErr
		return
	}

	for _, courseAccessRelation := range courseAccessRelations {
		student, getStudentErr := p.usersGateway.GetStudentById(courseAccessRelation.ObjectId)
		if getStudentErr != nil {
			err = getStudentErr
			return
		}
		students = append(students, student)
	}
	return
}

func (p *CourseUseCaseImpl) GetAccessCourseRelations(courseId string, parameterId string, parameter string) (courseRelations []*models.CourseRelationCore, err error) {
	return nil, errors.New("GetAccessCourseRelations is not implemented")
}

type CourseUseCaseModule struct {
	fx.Out
	courses.UseCase
}

func SetupCourseUseCase(
	gateway courses.Gateway,
	usersGateway users.Gateway,
	robboUnitGateway robboUnits.Gateway,
	robboGroupGateway robboGroup.Gateway,
) CourseUseCaseModule {
	return CourseUseCaseModule{
		UseCase: &CourseUseCaseImpl{
			courseGateway:     gateway,
			usersGateway:      usersGateway,
			robboUnitGateway:  robboUnitGateway,
			robboGroupGateway: robboGroupGateway,
		},
	}
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsStudents() (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsStudents()
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsTeachers() (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsTeachers()
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsUnitAdmins() (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsUnitAdmins()
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsRobboUnits() (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsRobboUnits()
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsRobboGroups() (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsRobboGroups()
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByStudentId(studentId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByStudentId(studentId)
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByTeacherId(teacherId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByTeacherId(teacherId)
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByUnitAdminId(unitAdminId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByUnitAdminId(unitAdminId)
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByRobboGroupId(robboGroupId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByRobboGroupId(robboGroupId)
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByRobboUnitId(robboUnitId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByRobboUnitId(robboUnitId)
}

func (p *CourseUseCaseImpl) GetAccessCourseRelationsByCourseId(courseId string) (courseRelations []*models.CourseRelationCore, err error) {
	return p.courseGateway.GetAccessCourseRelationsByCourseId(courseId)
}

func (p *CourseUseCaseImpl) CreateAccessCourseRelationRobboGroup(courseRelation *models.CourseRelationCore) (
	newCourseRelation *models.CourseRelationCore,
	err error,
) {
	//courseRelation.Parameter = "robbo_group"
	//return p.courseGateway.CreateAccessCourseRelation(courseRelation)

	// access for robbo group give access for teachers and students having relations with this robbo group (sorry my english)
	courseRelation.Parameter = "robbo_group"
	newCourseRelation, createAccessCourseRelationErr := p.courseGateway.CreateAccessCourseRelation(courseRelation)
	if createAccessCourseRelationErr != nil {
		err = createAccessCourseRelationErr
		return
	}

	// return relations between robbo group and teachers
	teachersRobboGroupRelations, getTeachersRelationErr := p.robboGroupGateway.GetRelationByRobboGroupId(courseRelation.ObjectId)
	if getTeachersRelationErr != nil {
		err = getTeachersRelationErr
		return
	}

	// return relations between robbo group and students
	studentsRobboGroupRelations, getStudentsRelationErr := p.usersGateway.GetStudentsByRobboGroupId(courseRelation.ObjectId)
	if getStudentsRelationErr != nil {
		err = getStudentsRelationErr
		return
	}

	// creating course access relations for unit admins
	for _, teachersRobboGroupRelation := range teachersRobboGroupRelations {
		newTeacherRelation := models.CourseRelationCore{
			ObjectId:  teachersRobboGroupRelation.TeacherId,
			Parameter: "teacher",
			CourseId:  courseRelation.CourseId,
		}
		_, createRelationErr := p.courseGateway.CreateAccessCourseRelation(&newTeacherRelation)
		if createRelationErr != nil {
			err = createRelationErr
			return
		}
	}

	// creating course access relations for students
	for _, studentsRobboGroupRelation := range studentsRobboGroupRelations {
		newStudentRelation := models.CourseRelationCore{
			ObjectId:  studentsRobboGroupRelation.UserCore.Id,
			Parameter: "student",
			CourseId:  courseRelation.CourseId,
		}
		_, createRelationErr := p.courseGateway.CreateAccessCourseRelation(&newStudentRelation)
		if createRelationErr != nil {
			err = createRelationErr
			return
		}
	}
	return
}

func (p *CourseUseCaseImpl) CreateAccessCourseRelationRobboUnit(courseRelation *models.CourseRelationCore) (
	newCourseRelation *models.CourseRelationCore,
	err error,
) {
	// access for robbo unit give access for unit admin having relations with this robbo unit (sorry my english)
	courseRelation.Parameter = "robbo_unit"
	newCourseRelation, createAccessCourseRelationErr := p.courseGateway.CreateAccessCourseRelation(courseRelation)
	if createAccessCourseRelationErr != nil {
		err = createAccessCourseRelationErr
		return
	}

	// return relations between robbo unit and unit admins
	unitAdminsRobboUnitRelations, getRelationErr := p.usersGateway.GetRelationByRobboUnitId(courseRelation.ObjectId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}

	// creating course access relations for unit admins
	for _, unitAdminsRobboUnitRelation := range unitAdminsRobboUnitRelations {
		newUnitAdminRelation := models.CourseRelationCore{
			ObjectId:  unitAdminsRobboUnitRelation.UnitAdminId,
			Parameter: "unit_admin",
			CourseId:  courseRelation.CourseId,
		}
		_, createRelationErr := p.courseGateway.CreateAccessCourseRelation(&newUnitAdminRelation)
		if createRelationErr != nil {
			err = createRelationErr
			return
		}
	}
	return
}

func (p *CourseUseCaseImpl) CreateAccessCourseRelationStudent(courseRelation *models.CourseRelationCore) (newCourseRelation *models.CourseRelationCore, err error) {
	courseRelation.Parameter = "student"
	return p.courseGateway.CreateAccessCourseRelation(courseRelation)
}

func (p *CourseUseCaseImpl) CreateAccessCourseRelationTeacher(courseRelation *models.CourseRelationCore) (newCourseRelation *models.CourseRelationCore, err error) {
	courseRelation.Parameter = "teacher"
	return p.courseGateway.CreateAccessCourseRelation(courseRelation)
}

func (p *CourseUseCaseImpl) CreateAccessCourseRelationUnitAdmin(courseRelation *models.CourseRelationCore) (newCourseRelation *models.CourseRelationCore, err error) {
	courseRelation.Parameter = "unit_admin"
	return p.courseGateway.CreateAccessCourseRelation(courseRelation)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationById(courseRelationId string) (id string, err error) {
	return p.courseGateway.DeleteAccessCourseRelationById(courseRelationId)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationsByStudentId(studentId string) (err error) {
	return p.courseGateway.DeleteAccessCourseRelationsByStudentId(studentId)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationsByTeacherId(teacherId string) (err error) {
	return p.courseGateway.DeleteAccessCourseRelationsByTeacherId(teacherId)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationsByUnitAdminId(unitAdminId string) (err error) {
	return p.courseGateway.DeleteAccessCourseRelationsByUnitAdminId(unitAdminId)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationsByRobboGroupId(robboGroupId string) (err error) {
	return p.courseGateway.DeleteAccessCourseRelationsByRobboGroupId(robboGroupId)
}

func (p *CourseUseCaseImpl) DeleteAccessCourseRelationsByRobboUnitId(robboUnitId string) (err error) {
	return p.courseGateway.DeleteAccessCourseRelationsByRobboUnitId(robboUnitId)
}

func (p *CourseUseCaseImpl) CreateCourse(course *models.CourseCore) (id string, err error) {
	CourseId, err := p.courseGateway.CreateCourse(course)
	if err != nil {
		log.Println("Error create Course")
		return "", err
	}

	mediaCore := &models.CourseApiMediaCollectionCore{
		CourseID: CourseId,
	}
	MediaId, err := p.courseGateway.CreateCourseApiMediaCollection(mediaCore)
	if err != nil {
		log.Println("Error create CourseApiMediaCollection")
		return "", err
	}

	bannerImage := &models.AbsoluteMediaCore{
		Uri:                        course.Media.BannerImage.Uri,
		UriAbsolute:                course.Media.BannerImage.UriAbsolute,
		CourseApiMediaCollectionID: MediaId,
	}
	_, err = p.courseGateway.CreateAbsoluteMedia(bannerImage)
	if err != nil {
		log.Println("Error create AbsoluteMedia")
		return "", err
	}

	courseImage := &models.MediaCore{
		Uri:                        course.Media.CourseImage.Uri,
		CourseApiMediaCollectionID: MediaId,
	}
	_, err = p.courseGateway.CreateMedia(courseImage)
	if err != nil {
		log.Println("Error create Media")
		return "", err
	}

	courseVideo := &models.MediaCore{
		Uri:                        course.Media.CourseVideo.Uri,
		CourseApiMediaCollectionID: MediaId,
	}
	_, err = p.courseGateway.CreateMedia(courseVideo)
	if err != nil {
		log.Println("Error create Media")
		return "", err
	}

	image := &models.ImageCore{
		Raw:                        course.Media.Image.Raw,
		Small:                      course.Media.Image.Small,
		Large:                      course.Media.Image.Large,
		CourseApiMediaCollectionID: MediaId,
	}
	_, err = p.courseGateway.CreateImage(image)
	if err != nil {
		log.Println("Error create Image")
		return "", err
	}

	return CourseId, nil
}

func (p *CourseUseCaseImpl) UpdateCourse(course *models.CourseCore) (err error) {
	err = p.courseGateway.UpdateAbsoluteMedia(&course.Media.BannerImage)
	if err != nil {
		log.Println("Error update AbsoluteMedia")
		return
	}

	err = p.courseGateway.UpdateMedia(&course.Media.CourseImage)
	if err != nil {
		log.Println("Error update Media")
		return
	}

	err = p.courseGateway.UpdateMedia(&course.Media.CourseVideo)
	if err != nil {
		log.Println("Error update Media")
		return
	}

	err = p.courseGateway.UpdateImage(&course.Media.Image)
	if err != nil {
		log.Println("Error update Image")
		return
	}

	err = p.courseGateway.UpdateCourseApiMediaCollection(&course.Media)
	if err != nil {
		log.Println("Error update CourseApiMediaCollection")
		return
	}

	err = p.courseGateway.UpdateCourse(course)
	if err != nil {
		log.Println("Error update Course")
		return
	}

	return nil
}

func (p *CourseUseCaseImpl) DeleteCourse(courseId string) (err error) {
	id, err := p.courseGateway.DeleteCourse(courseId)
	if err != nil {
		log.Println("Error delete Course")
		return
	}

	courseApiMediaCollectionId, err := p.courseGateway.DeleteCourseApiMediaCollection(id)
	if err != nil {
		log.Println("Error delete CourseApiMediaCollection")
		return
	}

	err = p.courseGateway.DeleteAbsoluteMedia(courseApiMediaCollectionId)
	if err != nil {
		log.Println("Error delete AbsoluteMedia")
		return
	}

	err = p.courseGateway.DeleteMedia(courseApiMediaCollectionId)
	if err != nil {
		log.Println("Error delete Media")
		return
	}

	err = p.courseGateway.DeleteImage(courseApiMediaCollectionId)
	if err != nil {
		log.Println("Error delete Image")
		return
	}

	return nil
}
