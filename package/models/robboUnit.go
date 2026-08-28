package models

import (
	"gorm.io/gorm"
	"strconv"
)

type RobboUnitCore struct {
	Id           string
	LastModified string
	Name         string
	City         string
}

type RobboUnitDB struct {
	gorm.Model

	Name string `gorm:"size:256;not null"`
	City string `gorm:"size:256;not null"`
}

func (em *RobboUnitDB) ToCore() *RobboUnitCore {
	return &RobboUnitCore{
		Id:           strconv.FormatUint(uint64(em.ID), 10),
		LastModified: RFC3339UTC(em.UpdatedAt),
		Name:         em.Name,
		City:         em.City,
	}
}

func (em *RobboUnitDB) FromCore(robboUnit *RobboUnitCore) {
	id, _ := strconv.ParseUint(robboUnit.Id, 10, 64)
	em.ID = uint(id)
	em.Name = robboUnit.Name
	em.City = robboUnit.City
}

func (ht *RobboUnitHTTP) ToCore() *RobboUnitCore {
	return &RobboUnitCore{
		Id:           ht.ID,
		LastModified: ht.LastModified,
		Name:         ht.Name,
		City:         ht.City,
	}
}

func (ht *RobboUnitHTTP) FromCore(robboUnit *RobboUnitCore) {
	ht.ID = robboUnit.Id
	ht.LastModified = robboUnit.LastModified
	ht.Name = robboUnit.Name
	ht.City = robboUnit.City
}
