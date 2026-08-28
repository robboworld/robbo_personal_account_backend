package models

import (
	"gorm.io/gorm"
	"strconv"
)

type RobboGroupCore struct {
	Id           string
	LastModified string
	Name         string
	RobboUnitId  string
	Students     []*StudentCore
}

type RobboGroupDB struct {
	gorm.Model
	RobboUnitId string `gorm:"not null"`
	Name        string `gorm:"size:256;not null"`
}

func (em *RobboGroupDB) ToCore() *RobboGroupCore {
	return &RobboGroupCore{
		Id:           strconv.FormatUint(uint64(em.ID), 10),
		LastModified: RFC3339UTC(em.UpdatedAt),
		Name:         em.Name,
		RobboUnitId:  em.RobboUnitId,
	}
}

func (em *RobboGroupDB) FromCore(robboGroup *RobboGroupCore) {
	id, _ := strconv.ParseUint(robboGroup.Id, 10, 64)
	em.ID = uint(id)
	em.Name = robboGroup.Name
	em.RobboUnitId = robboGroup.RobboUnitId
}

func (ht *RobboGroupHTTP) ToCore() *RobboGroupCore {
	var studentsCore []*StudentCore
	for _, studentHttp := range ht.Students {
		studentsCore = append(studentsCore, studentHttp.ToCore())
	}
	return &RobboGroupCore{
		Id:           ht.ID,
		LastModified: ht.LastModified,
		RobboUnitId:  ht.RobboUnitID,
		Name:         ht.Name,
		Students:     studentsCore,
	}
}

func (ht *RobboGroupHTTP) FromCore(robboGroup *RobboGroupCore) {
	ht.ID = robboGroup.Id
	ht.LastModified = robboGroup.LastModified
	ht.Name = robboGroup.Name
	ht.RobboUnitID = robboGroup.RobboUnitId
	for _, studentCore := range robboGroup.Students {
		studentHttpTemp := StudentHTTP{
			UserHTTP:     &UserHTTP{},
			RobboGroupID: "",
			RobboUnitID:  "",
		}
		studentHttpTemp.FromCore(studentCore)
		ht.Students = append(ht.Students, &studentHttpTemp)
	}

}
