package users

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Gateway interface {
	GetStudent(email, password string) (student *models.StudentCore, err error)
	SearchStudentsByEmail(email string, page, pageSize int) (students []*models.StudentCore, countRows int64, err error)
	AddStudentToRobboGroup(studentId, robboGroupId, robboUnitId string) (err error)
	CreateStudent(student *models.StudentCore) (newStudent *models.StudentCore, err error)
	UpdateStudent(student *models.StudentCore) (studentUpdated *models.StudentCore, err error)
	DeleteStudent(studentId string) (err error)
	GetStudentById(studentId string) (student *models.StudentCore, err error)
	GetStudentsByRobboGroupId(robboGroupId string) (students []*models.StudentCore, err error)
	GetStudentsByRobboUnitId(robboUnitId string) (students []*models.StudentCore, err error)

	CreateTeacher(teacher *models.TeacherCore) (newTeacher models.TeacherCore, err error)
	UpdateTeacher(teacher *models.TeacherCore) (teacherUpdated models.TeacherCore, err error)
	DeleteTeacher(teacherId string) (err error)
	GetTeacher(email, password string) (teacher models.TeacherCore, err error)
	GetAllTeachers(page, pageSize int) (teachers []models.TeacherCore, countRows int64, err error)
	GetTeacherById(teacherId string) (teacher models.TeacherCore, err error)
	SearchTeacherByEmail(email string, page, pageSize int) (teachers []models.TeacherCore, countRows int64, err error)

	CreateParent(parent *models.ParentCore) (newParent *models.ParentCore, err error)
	UpdateParent(parent *models.ParentCore) (parentUpdated *models.ParentCore, err error)
	DeleteParent(parentId string) (err error)
	GetParent(email, password string) (parent *models.ParentCore, err error)
	GetAllParent(page, pageSize int) (parents []*models.ParentCore, countRows int64, err error)
	GetParentById(parentId string) (parent *models.ParentCore, err error)

	CreateFreeListener(freeListener *models.FreeListenerCore) (newFreeListener *models.FreeListenerCore, err error)
	UpdateFreeListener(freeListener *models.FreeListenerCore) (freeListenerUpdated *models.FreeListenerCore, err error)
	DeleteFreeListener(freeListenerId string) (err error)
	GetFreeListener(email, password string) (freeListener *models.FreeListenerCore, err error)
	GetFreeListenerById(freeListenerId string) (freeListener *models.FreeListenerCore, err error)

	CreateUnitAdmin(unitAdmin *models.UnitAdminCore) (newUnitAdmin *models.UnitAdminCore, err error)
	UpdateUnitAdmin(unitAdmin *models.UnitAdminCore) (unitAdminUpdated *models.UnitAdminCore, err error)
	DeleteUnitAdmin(superAdminId string) (err error)
	GetUnitAdmin(email, password string) (unitAdmin *models.UnitAdminCore, err error)
	GetAllUnitAdmins(page, pageSize int) (unitAdmins []*models.UnitAdminCore, countRows int64, err error)
	GetUnitAdminById(unitAdminId string) (unitAdmin *models.UnitAdminCore, err error)
	SearchUnitAdminByEmail(email string, page, pageSize int) (unitAdmins []*models.UnitAdminCore, countRows int64, err error)

	UpdateSuperAdmin(superAdmin *models.SuperAdminCore) (superAdminUpdated *models.SuperAdminCore, err error)
	DeleteSuperAdmin(superAdminId string) (err error)
	GetSuperAdmin(email, password string) (superAdmin *models.SuperAdminCore, err error)
	GetSuperAdminById(superAdminId string) (superAdmin *models.SuperAdminCore, err error)

	SetUserAvatar(userID, avatarID string) (user models.UserCore, err error)

	CreateStudentParentRelation(relation *models.ChildrenOfParentCore) (err error)
	DeleteRelationByParentId(parentId string) (err error)
	DeleteRelationByChildrenId(childrenId string) (err error)
	DeleteRelation(relation *models.ChildrenOfParentCore) (err error)
	GetRelationByParentId(parentId string) (relations []*models.ChildrenOfParentCore, err error)
	GetRelationByChildrenId(childrenId string) (relations []*models.ChildrenOfParentCore, err error)

	SetUnitAdminForRobboUnit(relation *models.UnitAdminsRobboUnitsCore) (err error)
	DeleteUnitAdminForRobboUnit(relation *models.UnitAdminsRobboUnitsCore) (err error)
	DeleteRelationByRobboUnitId(robboUnitId string) (err error)
	DeleteRelationByUnitAdminId(unitAdminId string) (err error)
	GetRelationByRobboUnitId(robboUnitId string) (relations []*models.UnitAdminsRobboUnitsCore, err error)
	GetRelationByUnitAdminId(unitAdminId string, page, pageSize int) (relations []*models.UnitAdminsRobboUnitsCore, countRows int64, err error)

	CreateStudentTeacherRelation(relation *models.StudentsOfTeacherCore) (err error)
	DeleteStudentTeacherRelation(relation *models.StudentsOfTeacherCore) (err error)
	DeleteStudentTeacherRelationByTeacherId(teacherId string) (err error)
	DeleteStudentTeacherRelationByStudentId(studentId string) (err error)
	GetStudentTeacherRelationsByTeacherId(teacherId string) (relations []*models.StudentsOfTeacherCore, err error)
	GetStudentTeacherRelationsByStudentId(studentId string) (relations []*models.StudentsOfTeacherCore, err error)
}
