package models

import (
	"gorm.io/gorm"
	"strconv"
)

type TeacherCore struct {
	UserCore
}

type TeacherDB struct {
	gorm.Model
	UserDB
}

type TeacherHTTP struct {
	UserHTTP `json:"userHttp"`
}

func (em *TeacherDB) ToCore() TeacherCore {
	return TeacherCore{
		UserCore: em.UserDB.ToCore(),
	}
}

func (em *TeacherDB) FromCore(teacher *TeacherCore) {
	id, _ := strconv.ParseUint(teacher.Id, 10, 64)
	em.ID = uint(id)
	em.UserDB.FromCore(&teacher.UserCore)
}

func (ht *TeacherHTTP) ToCore() *TeacherCore {
	return &TeacherCore{
		UserCore: ht.UserHTTP.ToCore(),
	}
}

func (ht *TeacherHTTP) FromCore(teacher *TeacherCore) {
	ht.UserHTTP.FromCore(&teacher.UserCore)
}
