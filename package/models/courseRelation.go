package models

import (
	"gorm.io/gorm"
	"strconv"
)

type CourseRelationCore struct {
	Id           string
	LastModified string
	//параметр строка, который позволяет понять с чем связан курс (параметры: group, unit, role)
	Parameter string
	CourseId  string
	//id того с чем связь
	ObjectId string
}

type CourseRelationDB struct {
	gorm.Model

	Parameter string
	CourseId  string
	//Course    CourseDB `gorm:"foreignKey:CourseId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ObjectId string
}

func (em *CourseRelationDB) ToCore() *CourseRelationCore {
	return &CourseRelationCore{
		Id:           strconv.FormatUint(uint64(em.ID), 10),
		LastModified: RFC3339UTC(em.UpdatedAt),
		Parameter:    em.Parameter,
		CourseId:     em.CourseId,
		ObjectId:     em.ObjectId,
	}
}

func (em *CourseRelationDB) FromCore(core *CourseRelationCore) {
	id, _ := strconv.ParseUint(core.Id, 10, 64)
	em.ID = uint(id)
	em.Parameter = core.Parameter
	em.CourseId = core.CourseId
	em.ObjectId = core.ObjectId
}

func (ht *CourseRelationHTTP) ToCore() *CourseRelationCore {
	return &CourseRelationCore{
		Id:       ht.ID,
		CourseId: ht.CourseID,
		ObjectId: ht.ObjectID,
	}
}

func (ht *CourseRelationHTTP) FromCore(core *CourseRelationCore) {
	ht.ID = core.Id
	ht.LastModified = core.LastModified
	ht.Parameter = core.Parameter
	ht.CourseID = core.CourseId
	ht.ObjectID = core.ObjectId
}
