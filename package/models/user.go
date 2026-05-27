package models

import (
	"github.com/dgrijalva/jwt-go/v4"
	"gorm.io/gorm"
	"strconv"
)

type Role int

const (
	Student Role = iota
	Teacher
	Parent
	FreeListener
	UnitAdmin
	SuperAdmin
	Anonymous
	WithExpiredToken
)

type UserClaims struct {
	jwt.StandardClaims
	Id   string
	Role Role
}

type UserDB struct {
	gorm.Model
	Email      string `gorm:"not null;size:256;uniqueIndex"`
	Password   string `gorm:"not null;size:256"`
	Role       uint   `gorm:"not null"`
	Nickname   string `gorm:"not null;size:256"`
	Firstname  string `gorm:"not null;size:256"`
	Middlename string `gorm:"not null;size:256"`
	Lastname   string `gorm:"not null;size:256"`
}

type UserCore struct {
	Id               string
	Email            string
	Password         string
	Role             Role
	Nickname         string
	FullName         string
	Firstname        string
	Middlename       string
	Lastname         string
	Company          string
	LevelOfEducation string
	Country          string
	YearOfBirth      *int
	Gender           string
	Language         string
	CreatedAt        string
}

func (em *UserHTTP) ToCore() UserCore {
	return UserCore{
		Id:               em.ID,
		Email:            em.Email,
		Password:         em.Password,
		Role:             Role(em.Role),
		Nickname:         em.Nickname,
		FullName:         em.FullName,
		Firstname:        em.Firstname,
		Lastname:         em.Lastname,
		Middlename:       em.Middlename,
		Company:          em.Company,
		LevelOfEducation: StrPtrVal(em.LevelOfEducation),
		Country:          StrPtrVal(em.Country),
		YearOfBirth:      em.YearOfBirth,
		Gender:           StrPtrVal(em.Gender),
		Language:         StrPtrVal(em.Language),
		CreatedAt:        em.CreatedAt,
	}
}

func (em *UserHTTP) FromCore(user *UserCore) {
	em.ID = user.Id
	em.Email = user.Email
	em.Password = user.Password
	em.Role = int(user.Role)
	em.Nickname = user.Nickname
	em.FullName = user.FullName
	em.Firstname = user.Firstname
	em.Lastname = user.Lastname
	em.Middlename = user.Middlename
	em.Company = user.Company
	em.LevelOfEducation = StrPtr(user.LevelOfEducation)
	em.Country = StrPtr(user.Country)
	em.YearOfBirth = user.YearOfBirth
	em.Gender = StrPtr(user.Gender)
	em.Language = StrPtr(user.Language)
	em.CreatedAt = user.CreatedAt
}

func (em *UserDB) ToCore() UserCore {
	return UserCore{
		Id:         strconv.FormatUint(uint64(em.ID), 10),
		Email:      em.Email,
		Password:   em.Password,
		Role:       Role(em.Role),
		Nickname:   em.Nickname,
		Firstname:  em.Firstname,
		Lastname:   em.Lastname,
		Middlename: em.Middlename,
		CreatedAt:  em.CreatedAt.String(),
	}
}

func (em *UserDB) FromCore(user *UserCore) {
	id, _ := strconv.ParseUint(user.Id, 10, 64)
	em.ID = uint(id)
	em.Email = user.Email
	em.Password = user.Password
	em.Role = uint(user.Role)
	em.Nickname = user.Nickname
	em.Firstname = user.Firstname
	em.Lastname = user.Lastname
	em.Middlename = user.Middlename
}
