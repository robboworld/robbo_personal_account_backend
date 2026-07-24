package models

import (
	"gorm.io/gorm"
	"strconv"
)

type ProjectPageCore struct {
	ProjectPageId    string
	LastModified     string
	Title            string
	ProjectId        string
	Instruction      string
	Notes            string
	Preview          string
	LinkScratch      string
	IsShared         bool
	LandingFeatured  bool
	LandingSortOrder int
	AuthorUserId     string
	AuthorName       string
	IsOwner          bool
}

type ProjectPageDB struct {
	gorm.Model

	ProjectId   string
	Project     ProjectDB `gorm:"foreignKey:ProjectId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Instruction string    `gorm:"size:256;not null"`
	Notes       string    `gorm:"size:256;not null"`
	Preview     string    `gorm:"size:256;not null"`
	LinkScratch string    `gorm:"size:256;not null"`
	Title       string    `gorm:"size:256;not null"`
	IsShared    bool
}

func (em *ProjectPageDB) ToCore() *ProjectPageCore {
	return &ProjectPageCore{
		ProjectPageId: strconv.FormatUint(uint64(em.ID), 10),
		LastModified:  em.UpdatedAt.String(),
		Title:         em.Title,
		ProjectId:     em.ProjectId,
		Instruction:   em.Instruction,
		Notes:         em.Notes,
		Preview:       em.Preview,
		LinkScratch:   em.LinkScratch,
		IsShared:      em.IsShared,
	}
}

func (em *ProjectPageDB) FromCore(projectPage *ProjectPageCore) {
	id, _ := strconv.ParseUint(projectPage.ProjectPageId, 10, 64)
	em.ID = uint(id)
	em.ProjectId = projectPage.ProjectId
	em.Instruction = projectPage.Instruction
	em.Notes = projectPage.Notes
	em.Preview = projectPage.Preview
	em.LinkScratch = projectPage.LinkScratch
	em.Title = projectPage.Title
	em.IsShared = projectPage.IsShared
}

func (ht *ProjectPageHTTP) ToCore() *ProjectPageCore {
	return &ProjectPageCore{
		ProjectPageId:    ht.ProjectPageID,
		LastModified:     ht.LastModified,
		Title:            ht.Title,
		ProjectId:        ht.ProjectID,
		Instruction:      ht.Instruction,
		Notes:            ht.Notes,
		Preview:          ht.Preview,
		LinkScratch:      ht.LinkScratch,
		IsShared:         ht.IsShared,
		LandingFeatured:  ht.LandingFeatured,
		LandingSortOrder: ht.LandingSortOrder,
		AuthorUserId:     ht.AuthorUserID,
		AuthorName:       ht.AuthorName,
		IsOwner:          ht.IsOwner,
	}
}

func (ht *ProjectPageHTTP) FromCore(projectPage *ProjectPageCore) {
	ht.ProjectPageID = projectPage.ProjectPageId
	ht.LastModified = projectPage.LastModified
	ht.ProjectID = projectPage.ProjectId
	ht.Instruction = projectPage.Instruction
	ht.Notes = projectPage.Notes
	ht.Preview = projectPage.Preview
	ht.LinkScratch = projectPage.LinkScratch
	ht.Title = projectPage.Title
	ht.IsShared = projectPage.IsShared
	ht.LandingFeatured = projectPage.LandingFeatured
	ht.LandingSortOrder = projectPage.LandingSortOrder
	ht.AuthorUserID = projectPage.AuthorUserId
	ht.AuthorName = projectPage.AuthorName
	ht.IsOwner = projectPage.IsOwner
}
