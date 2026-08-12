package usecase

import (
	"crypto/sha1"
	"fmt"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type UsersUseCaseImpl struct {
	usersGateway      users.Gateway
	robboGroupGateway robboGroup.Gateway
}

type UsersUseCaseModule struct {
	fx.Out
	users.UseCase
}

func SetupUsersUseCase(usersGateway users.Gateway, robboGroupGateway robboGroup.Gateway) UsersUseCaseModule {
	return UsersUseCaseModule{
		UseCase: &UsersUseCaseImpl{
			usersGateway:      usersGateway,
			robboGroupGateway: robboGroupGateway,
		},
	}
}

func (p *UsersUseCaseImpl) GetPairsStudentParentsByTeacherId(teacherId string) (pairsStudentParents []*models.StudentParentsCore, err error) {
	studentTeacherRelations, getStudentTeacherRelationsErr := p.usersGateway.GetStudentTeacherRelationsByTeacherId(teacherId)
	if getStudentTeacherRelationsErr != nil {
		err = getStudentTeacherRelationsErr
		return
	}
	for _, studentTeacherRelation := range studentTeacherRelations {
		student, getStudentErr := p.usersGateway.GetStudentById(studentTeacherRelation.StudentId)
		if getStudentErr != nil {
			err = getStudentErr
			return
		}
		studentParentRelations, getRelationByChildrenIdErr := p.usersGateway.GetRelationByChildrenId(student.Id)
		if getRelationByChildrenIdErr != nil {
			err = getRelationByChildrenIdErr
			return
		}
		var parents []*models.ParentCore
		for _, studentParentRelation := range studentParentRelations {
			parent, getParentErr := p.usersGateway.GetParentById(studentParentRelation.ParentId)
			if getParentErr != nil {
				err = getParentErr
				return
			}
			parents = append(parents, parent)
		}
		pairStudentParentsTemp := &models.StudentParentsCore{
			Student: student,
			Parents: parents,
		}
		pairsStudentParents = append(pairsStudentParents, pairStudentParentsTemp)
	}
	return
}

func (p *UsersUseCaseImpl) GetIndividualStudentsByTeacherId(teacherId string) (students []*models.StudentCore, err error) {
	relations, getRelationErr := p.usersGateway.GetStudentTeacherRelationsByTeacherId(teacherId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}
	for _, relation := range relations {
		student, getStudentErr := p.usersGateway.GetStudentById(relation.StudentId)
		if getStudentErr != nil {
			err = getStudentErr
			return
		}
		if student.RobboUnitId != "0" {
			continue
		}
		if student.RobboGroupId != "0" {
			continue
		}
		students = append(students, student)
	}
	return
}

func (p *UsersUseCaseImpl) GetStudentsByRobboUnitId(robboUnitId string) (students []*models.StudentCore, err error) {
	return p.usersGateway.GetStudentsByRobboUnitId(robboUnitId)
}

func (p *UsersUseCaseImpl) GetStudentsByTeacherId(teacherId string) (students []*models.StudentCore, err error) {
	relations, getRelationErr := p.usersGateway.GetStudentTeacherRelationsByTeacherId(teacherId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}
	for _, relation := range relations {
		student, getTeacherErr := p.usersGateway.GetStudentById(relation.StudentId)
		if getTeacherErr != nil {
			err = getTeacherErr
			return
		}
		students = append(students, student)
	}
	return
}

func (p *UsersUseCaseImpl) GetTeacherByRobboGroupId(robboGroupId string) (teachers []*models.TeacherCore, err error) {
	relations, getRelationErr := p.robboGroupGateway.GetRelationByRobboGroupId(robboGroupId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}
	for _, relation := range relations {
		teacher, getTeacherErr := p.usersGateway.GetTeacherById(relation.TeacherId)
		if getTeacherErr != nil {
			err = getTeacherErr
			return
		}
		teachers = append(teachers, &teacher)
	}
	return
}

func (p *UsersUseCaseImpl) GetStudentsByRobboGroupId(robboGroupId string) (students []*models.StudentCore, err error) {
	return p.usersGateway.GetStudentsByRobboGroupId(robboGroupId)
}

//func (p *UsersUseCaseImpl) GetStudent(email, password string) (student *models.StudentCore, err error) {
//	return p.usersGateway.GetStudent(email, password)
//}

func (p *UsersUseCaseImpl) GetStudentById(studentId string) (student *models.StudentCore, err error) {
	return p.usersGateway.GetStudentById(studentId)
}

func (p *UsersUseCaseImpl) SearchStudentByEmail(email string, page, pageSize int) (
	students []*models.StudentCore,
	countRows int64,
	err error,
) {
	emailCondition := email + "%"
	return p.usersGateway.SearchStudentsByEmail(emailCondition, page, pageSize)
}

func (p *UsersUseCaseImpl) GetStudentByParentId(parentId string) (students []*models.StudentCore, err error) {
	relations, getRelationErr := p.usersGateway.GetRelationByParentId(parentId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}
	for _, relation := range relations {
		student, getStudentErr := p.usersGateway.GetStudentById(relation.ChildId)
		if getStudentErr != nil {
			err = getStudentErr
			return
		}
		students = append(students, student)
	}
	return
}

func (p *UsersUseCaseImpl) CreateStudent(student *models.StudentCore, parentId string) (newStudent *models.StudentCore, err error) {
	pwd := sha1.New()
	pwd.Write([]byte(student.Password))
	pwd.Write([]byte(viper.GetString("auth.hash_salt")))
	passwordHash := fmt.Sprintf("%x", pwd.Sum(nil))
	student.Password = passwordHash
	newStudent, err = p.usersGateway.CreateStudent(student)
	if err != nil {
		return
	}
	relation := &models.ChildrenOfParentCore{
		ChildId:  newStudent.Id,
		ParentId: parentId,
	}
	if parentId != "" {
		err = p.usersGateway.CreateStudentParentRelation(relation)
	}
	return
}

func (p *UsersUseCaseImpl) DeleteStudent(studentId string) (err error) {
	if err = p.usersGateway.DeleteStudent(studentId); err != nil {
		return
	}
	if err = p.usersGateway.DeleteRelationByChildrenId(studentId); err != nil {
		return
	}
	if err = p.usersGateway.DeleteStudentTeacherRelationByStudentId(studentId); err != nil {
		return
	}
	return
}

func (p *UsersUseCaseImpl) UpdateStudent(student *models.StudentCore) (studentUpdated *models.StudentCore, err error) {
	return p.usersGateway.UpdateStudent(student)
}

func (p *UsersUseCaseImpl) AddStudentToRobboGroup(studentId string, robboGroupId string, robboUnitId string) (err error) {
	if err = p.usersGateway.AddStudentToRobboGroup(studentId, robboGroupId, robboUnitId); err != nil {
		return
	}
	teachersRobboGroupsRelations, err := p.robboGroupGateway.GetRelationByRobboGroupId(robboGroupId)
	if err != nil {
		return
	}
	for _, relation := range teachersRobboGroupsRelations {
		relationCore := &models.StudentsOfTeacherCore{
			StudentId: studentId,
			TeacherId: relation.TeacherId,
		}
		if err = p.usersGateway.CreateStudentTeacherRelation(relationCore); err != nil {
			return
		}
	}
	return
}

func (p *UsersUseCaseImpl) GetTeacherById(teacherId string) (teacher models.TeacherCore, err error) {
	return p.usersGateway.GetTeacherById(teacherId)
}

func (p *UsersUseCaseImpl) GetTeachersByStudentId(studentId string) (teachers []*models.TeacherCore, err error) {
	relations, getRelationErr := p.usersGateway.GetStudentTeacherRelationsByStudentId(studentId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}
	for _, relation := range relations {
		teacher, getTeacherErr := p.usersGateway.GetTeacherById(relation.TeacherId)
		if getTeacherErr != nil {
			err = getTeacherErr
			return
		}
		teachers = append(teachers, &teacher)
	}
	return
}

//func (p *UsersUseCaseImpl) GetTeacher(email, password string) (teacher *models.TeacherCore, err error) {
//	return p.usersGateway.GetTeacher(email, password)
//}

func (p *UsersUseCaseImpl) GetAllTeachers(page, pageSize int) (teachers []models.TeacherCore, countRows int64, err error) {
	return p.usersGateway.GetAllTeachers(page, pageSize)
}

func (p *UsersUseCaseImpl) UpdateTeacher(teacher *models.TeacherCore) (teacherUpdated models.TeacherCore, err error) {
	return p.usersGateway.UpdateTeacher(teacher)
}

func (p *UsersUseCaseImpl) CreateTeacher(teacher *models.TeacherCore) (newTeacher models.TeacherCore, err error) {
	pwd := sha1.New()
	pwd.Write([]byte(teacher.Password))
	pwd.Write([]byte(viper.GetString("auth.hash_salt")))
	passwordHash := fmt.Sprintf("%x", pwd.Sum(nil))
	teacher.Password = passwordHash
	return p.usersGateway.CreateTeacher(teacher)
}

func (p *UsersUseCaseImpl) DeleteTeacher(teacherId string) (err error) {
	if err = p.usersGateway.DeleteTeacher(teacherId); err != nil {
		return
	}
	if err = p.usersGateway.DeleteStudentTeacherRelationByTeacherId(teacherId); err != nil {
		return
	}
	return
}

func (p *UsersUseCaseImpl) SearchTeacherByEmail(email string, page, pageSize int) (
	teachers []models.TeacherCore,
	countRows int64,
	err error,
) {
	emailCondition := email + "%"
	return p.usersGateway.SearchTeacherByEmail(emailCondition, page, pageSize)
}

func (p *UsersUseCaseImpl) GetParentById(parentId string) (parent *models.ParentCore, err error) {
	return p.usersGateway.GetParentById(parentId)
}

func (p *UsersUseCaseImpl) GetAllParent(page, pageSize int) (parents []*models.ParentCore, countRows int64, err error) {
	return p.usersGateway.GetAllParent(page, pageSize)
}

func (p *UsersUseCaseImpl) CreateParent(parent *models.ParentCore) (newParent *models.ParentCore, err error) {
	pwd := sha1.New()
	pwd.Write([]byte(parent.Password))
	pwd.Write([]byte(viper.GetString("auth.hash_salt")))
	passwordHash := fmt.Sprintf("%x", pwd.Sum(nil))
	parent.Password = passwordHash
	return p.usersGateway.CreateParent(parent)
}

func (p *UsersUseCaseImpl) DeleteParent(parentId string) (err error) {
	relations, getRelationsErr := p.usersGateway.GetRelationByParentId(parentId)
	if getRelationsErr != nil {
		return getRelationsErr
	}

	for _, relation := range relations {
		deleteStudentErr := p.usersGateway.DeleteStudent(relation.ChildId)
		if deleteStudentErr != nil {
			return deleteStudentErr
		}
	}
	deleteRelationErr := p.usersGateway.DeleteRelationByParentId(parentId)
	if deleteRelationErr != nil {
		return deleteRelationErr
	}
	return p.usersGateway.DeleteParent(parentId)
}

func (p *UsersUseCaseImpl) UpdateParent(parent *models.ParentCore) (parentUpdated *models.ParentCore, err error) {
	return p.usersGateway.UpdateParent(parent)
}

//func (p *UsersUseCaseImpl) GetFreeListener(email, password string) (freeListener *models.FreeListenerCore, err error) {
//	return p.usersGateway.GetFreeListener(email, password)
//}

func (p *UsersUseCaseImpl) GetFreeListenerById(freeListenerId string) (freeListener *models.FreeListenerCore, err error) {
	return p.usersGateway.GetFreeListenerById(freeListenerId)
}

func (p *UsersUseCaseImpl) CreateFreeListener(freeListener *models.FreeListenerCore) (newFreeListener *models.FreeListenerCore, err error) {
	return p.usersGateway.CreateFreeListener(freeListener)
}

func (p *UsersUseCaseImpl) DeleteFreeListener(freeListenerId string) (err error) {
	return p.usersGateway.DeleteFreeListener(freeListenerId)
}

func (p *UsersUseCaseImpl) UpdateFreeListener(freeListener *models.FreeListenerCore) (freeListenerUpdated *models.FreeListenerCore, err error) {
	return p.usersGateway.UpdateFreeListener(freeListener)
}

func (p *UsersUseCaseImpl) GetUnitAdminById(unitAdminId string) (unitAdmin *models.UnitAdminCore, err error) {
	return p.usersGateway.GetUnitAdminById(unitAdminId)
}

func (p *UsersUseCaseImpl) GetAllUnitAdmins(page, pageSize int) (unitAdmins []*models.UnitAdminCore, countRows int64, err error) {
	return p.usersGateway.GetAllUnitAdmins(page, pageSize)
}

//func (p *UsersUseCaseImpl) GetUnitAdmin(email, password string) (unitAdmin *models.UnitAdminCore, err error) {
//	return p.usersGateway.GetUnitAdmin(email, password)
//}

func (p *UsersUseCaseImpl) UpdateUnitAdmin(unitAdmin *models.UnitAdminCore) (unitAdminUpdated *models.UnitAdminCore, err error) {
	return p.usersGateway.UpdateUnitAdmin(unitAdmin)
}

func (p *UsersUseCaseImpl) CreateUnitAdmin(unitAdmin *models.UnitAdminCore) (newUnitAdmin *models.UnitAdminCore, err error) {
	pwd := sha1.New()
	pwd.Write([]byte(unitAdmin.Password))
	pwd.Write([]byte(viper.GetString("auth.hash_salt")))
	passwordHash := fmt.Sprintf("%x", pwd.Sum(nil))
	unitAdmin.Password = passwordHash
	return p.usersGateway.CreateUnitAdmin(unitAdmin)
}

func (p *UsersUseCaseImpl) DeleteUnitAdmin(unitAdminId string) (err error) {
	return p.usersGateway.DeleteUnitAdmin(unitAdminId)
}

func (p *UsersUseCaseImpl) SearchUnitAdminByEmail(email string, page, pageSize int) (
	unitAdmins []*models.UnitAdminCore,
	countRows int64,
	err error,
) {
	emailCondition := email + "%"
	return p.usersGateway.SearchUnitAdminByEmail(emailCondition, page, pageSize)
}

func (p *UsersUseCaseImpl) GetUnitAdminByRobboUnitId(robboUnitId string) (unitAdmins []*models.UnitAdminCore, err error) {
	relations, getRelationErr := p.usersGateway.GetRelationByRobboUnitId(robboUnitId)
	if getRelationErr != nil {
		err = getRelationErr
		return
	}

	for _, relation := range relations {
		unitAdmin, getUnitAdminErr := p.usersGateway.GetUnitAdminById(relation.UnitAdminId)
		if getUnitAdminErr != nil {
			err = getRelationErr
			return
		}
		unitAdmins = append(unitAdmins, unitAdmin)
	}
	return
}

func (p *UsersUseCaseImpl) GetSuperAdminById(superAdminId string) (superAdmin *models.SuperAdminCore, err error) {
	return p.usersGateway.GetSuperAdminById(superAdminId)
}

func (p *UsersUseCaseImpl) UpdateSuperAdmin(superAdmin *models.SuperAdminCore) (superAdminUpdated *models.SuperAdminCore, err error) {
	return p.usersGateway.UpdateSuperAdmin(superAdmin)
}
func (p *UsersUseCaseImpl) DeleteSuperAdmin(superAdminId string) (err error) {
	return p.usersGateway.DeleteSuperAdmin(superAdminId)
}

func (p *UsersUseCaseImpl) SetUserAvatar(userID, avatarID string) (user models.UserCore, err error) {
	return p.usersGateway.SetUserAvatar(userID, avatarID)
}

func (p *UsersUseCaseImpl) CreateStudentParentRelation(parentId, childrenId string) (studentsCore []*models.StudentCore, err error) {
	relationCore := &models.ChildrenOfParentCore{
		ChildId:  childrenId,
		ParentId: parentId,
	}
	createRelationErr := p.usersGateway.CreateStudentParentRelation(relationCore)
	if createRelationErr != nil {
		err = createRelationErr
		return
	}

	studentsCore, err = p.GetStudentByParentId(parentId)
	return
}

func (p *UsersUseCaseImpl) SetNewUnitAdminForRobboUnit(unitAdminId, robboUnitId string) (err error) {
	relationCore := &models.UnitAdminsRobboUnitsCore{
		UnitAdminId: unitAdminId,
		RobboUnitId: robboUnitId,
	}
	return p.usersGateway.SetUnitAdminForRobboUnit(relationCore)
}

func (p *UsersUseCaseImpl) DeleteUnitAdminForRobboUnit(unitAdminId, robboUnitId string) (err error) {
	relationCore := &models.UnitAdminsRobboUnitsCore{
		UnitAdminId: unitAdminId,
		RobboUnitId: robboUnitId,
	}
	return p.usersGateway.DeleteUnitAdminForRobboUnit(relationCore)
}

func (p *UsersUseCaseImpl) CreateStudentTeacherRelation(studentId, teacherId string) (student *models.StudentCore, err error) {
	relationCore := &models.StudentsOfTeacherCore{
		StudentId: studentId,
		TeacherId: teacherId,
	}
	if createRelationErr := p.usersGateway.CreateStudentTeacherRelation(relationCore); createRelationErr != nil {
		err = createRelationErr
		return
	}
	student, getStudentErr := p.usersGateway.GetStudentById(studentId)
	if getStudentErr != nil {
		err = getStudentErr
		return
	}
	return
}

func (p *UsersUseCaseImpl) DeleteStudentTeacherRelation(studentId, teacherId string) (student *models.StudentCore, err error) {
	relationCore := &models.StudentsOfTeacherCore{
		StudentId: studentId,
		TeacherId: teacherId,
	}
	if deleteRelationErr := p.usersGateway.DeleteStudentTeacherRelation(relationCore); deleteRelationErr != nil {
		err = deleteRelationErr
		return
	}
	student, getStudentErr := p.usersGateway.GetStudentById(studentId)
	if getStudentErr != nil {
		err = getStudentErr
		return
	}
	return
}
