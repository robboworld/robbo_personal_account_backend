package users

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type UseCase interface {
	GetIndividualStudentsByTeacherId(teacherId string) (students []*models.StudentCore, err error)
	GetStudentById(studentId string) (student *models.StudentCore, err error)
	GetStudentByParentId(parentId string) (students []*models.StudentCore, err error)
	GetStudentsByRobboGroupId(robboGroupId string) (students []*models.StudentCore, err error)
	GetStudentsByRobboUnitId(robboUnitId string) (students []*models.StudentCore, err error)
	GetStudentsByTeacherId(teacherId string) (students []*models.StudentCore, err error)
	SearchStudentByEmail(email string, page, pageSize int) (students []*models.StudentCore, countRows int64, err error)
	CreateStudent(student *models.StudentCore, parentId string) (newStudent *models.StudentCore, err error)
	DeleteStudent(studentId string) (err error)
	UpdateStudent(student *models.StudentCore) (studentUpdated *models.StudentCore, err error)
	AddStudentToRobboGroup(studentId string, robboGroupId string, robboUnitId string) (err error)

	GetPairsStudentParentsByTeacherId(teacherId string) (pairsStudentParents []*models.StudentParentsCore, err error)

	GetTeacherById(teacherId string) (teacher models.TeacherCore, err error)
	GetAllTeachers(page, pageSize int) (teachers []models.TeacherCore, countRows int64, err error)
	CreateTeacher(teacher *models.TeacherCore) (newTeacher models.TeacherCore, err error)
	UpdateTeacher(teacher *models.TeacherCore) (teacherUpdated models.TeacherCore, err error)
	DeleteTeacher(teacherId string) (err error)
	GetTeacherByRobboGroupId(robboGroupId string) (teachers []*models.TeacherCore, err error)
	GetTeachersByStudentId(studentId string) (teachers []*models.TeacherCore, err error)
	SearchTeacherByEmail(email string, page, pageSize int) (teachers []models.TeacherCore, countRows int64, err error)

	GetParentById(parentId string) (parent *models.ParentCore, err error)
	GetAllParent(page, pageSize int) (parents []*models.ParentCore, countRows int64, err error)
	CreateParent(parent *models.ParentCore) (newParent *models.ParentCore, err error)
	UpdateParent(parent *models.ParentCore) (parentUpdated *models.ParentCore, err error)
	DeleteParent(parentId string) (err error)

	GetFreeListenerById(freeListenerId string) (freeListener *models.FreeListenerCore, err error)
	CreateFreeListener(freeListener *models.FreeListenerCore) (newFreeListener *models.FreeListenerCore, err error)
	UpdateFreeListener(freeListener *models.FreeListenerCore) (freeListenerUpdated *models.FreeListenerCore, err error)
	DeleteFreeListener(freeListenerId string) (err error)

	GetUnitAdminById(unitAdminId string) (unitAdmin *models.UnitAdminCore, err error)
	GetAllUnitAdmins(page, pageSize int) (unitAdmins []*models.UnitAdminCore, countRows int64, err error)
	GetUnitAdminByRobboUnitId(robboUnitId string) (unitAdmins []*models.UnitAdminCore, err error)
	CreateUnitAdmin(unitAdmin *models.UnitAdminCore) (newUnitAdmin *models.UnitAdminCore, err error)
	UpdateUnitAdmin(unitAdmin *models.UnitAdminCore) (unitAdminUpdated *models.UnitAdminCore, err error)
	DeleteUnitAdmin(unitAdminId string) (err error)
	SearchUnitAdminByEmail(email string, page, pageSize int) (unitAdmins []*models.UnitAdminCore, countRows int64, err error)

	GetSuperAdminById(superAdminId string) (superAdmin *models.SuperAdminCore, err error)
	UpdateSuperAdmin(superAdmin *models.SuperAdminCore) (superAdminUpdated *models.SuperAdminCore, err error)
	DeleteSuperAdmin(superAdminId string) (err error)

	SetUserAvatar(userID, avatarID string) (user models.UserCore, err error)

	CreateStudentParentRelation(parentId, childrenId string) (studentsHTTP []*models.StudentCore, err error)
	SetNewUnitAdminForRobboUnit(unitAdminId, robboUnitId string) (err error)
	DeleteUnitAdminForRobboUnit(unitAdminId, robboUnitId string) (err error)

	CreateStudentTeacherRelation(studentId, teacherId string) (student *models.StudentCore, err error)
	DeleteStudentTeacherRelation(studentId, teacherId string) (student *models.StudentCore, err error)
}
