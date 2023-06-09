package models

import (
	"gorm.io/gorm"
	"strconv"
)

type AbsoluteMediaCore struct {
	ID                         string
	Uri                        string
	UriAbsolute                string
	CourseApiMediaCollectionID string
}

type AbsoluteMediaDB struct {
	gorm.Model
	Uri                        string
	UriAbsolute                string
	CourseApiMediaCollectionID uint
	CourseApiMediaCollection   CourseApiMediaCollectionDB `gorm:"foreignKey:CourseApiMediaCollectionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (em *AbsoluteMediaDB) ToCore() *AbsoluteMediaCore {
	return &AbsoluteMediaCore{
		ID:                         strconv.FormatUint(uint64(em.ID), 10),
		Uri:                        em.Uri,
		UriAbsolute:                em.UriAbsolute,
		CourseApiMediaCollectionID: strconv.FormatUint(uint64(em.CourseApiMediaCollectionID), 10),
	}
}

func (em *AbsoluteMediaDB) FromCore(absoluteMedia *AbsoluteMediaCore) {
	id, _ := strconv.ParseUint(absoluteMedia.ID, 10, 64)
	courseApiMediaCollectionId, _ := strconv.ParseUint(absoluteMedia.CourseApiMediaCollectionID, 10, 64)
	em.ID = uint(id)
	em.Uri = absoluteMedia.Uri
	em.UriAbsolute = absoluteMedia.UriAbsolute
	em.CourseApiMediaCollectionID = uint(courseApiMediaCollectionId)
}

func (ht *AbsoluteMediaHTTP) ToCore() *AbsoluteMediaCore {
	return &AbsoluteMediaCore{
		ID:          ht.ID,
		Uri:         ht.URI,
		UriAbsolute: ht.URIAbsolute,
	}
}

func (ht *AbsoluteMediaHTTP) FromCore(absoluteMedia *AbsoluteMediaCore) {
	ht.ID = absoluteMedia.ID
	ht.URI = absoluteMedia.Uri
	ht.URIAbsolute = absoluteMedia.UriAbsolute
}
