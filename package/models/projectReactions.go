package models

import "time"

type ScratchReactionTypeDB struct {
	Code      string    `gorm:"column:code;primaryKey"`
	Emoji     string    `gorm:"column:emoji"`
	SortOrder int       `gorm:"column:sort_order"`
	IsEnabled bool      `gorm:"column:is_enabled"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ScratchReactionTypeDB) TableName() string { return "scratch_reaction_types" }

type ScratchProjectReactionDB struct {
	ProjectID    string    `gorm:"column:project_id;primaryKey"`
	UserID       string    `gorm:"column:user_id;primaryKey"`
	ReactionCode string    `gorm:"column:reaction_code"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ScratchProjectReactionDB) TableName() string { return "scratch_project_reactions" }

type ReactionTypeHTTP struct {
	Code  string `json:"code"`
	Emoji string `json:"emoji"`
}

type ReactionCountHTTP struct {
	Code  string `json:"code"`
	Emoji string `json:"emoji"`
	Count int64  `json:"count"`
}

type ProjectReactionsHTTP struct {
	Counts     []ReactionCountHTTP `json:"counts"`
	MyReaction *string             `json:"myReaction"`
}
