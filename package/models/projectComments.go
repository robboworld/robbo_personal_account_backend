package models

import "time"

// ScratchProjectCommentDB maps scratch_project_comments.
type ScratchProjectCommentDB struct {
	ID        string     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	ProjectID string     `gorm:"column:project_id"`
	UserID    string     `gorm:"column:user_id"`
	ParentID  *string    `gorm:"column:parent_id"`
	Body      string     `gorm:"column:body"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (ScratchProjectCommentDB) TableName() string { return "scratch_project_comments" }

// ScratchCommentReactionDB maps scratch_comment_reactions.
type ScratchCommentReactionDB struct {
	CommentID    string    `gorm:"column:comment_id;primaryKey"`
	UserID       string    `gorm:"column:user_id;primaryKey"`
	ReactionCode string    `gorm:"column:reaction_code"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ScratchCommentReactionDB) TableName() string { return "scratch_comment_reactions" }

// CommentAuthorHTTP is a compact author card on a comment.
type CommentAuthorHTTP struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	FullName string `json:"fullName"`
}

// CommentReactionsHTTP aggregates like/dislike on a comment.
type CommentReactionsHTTP struct {
	Likes      int64   `json:"likes"`
	Dislikes   int64   `json:"dislikes"`
	MyReaction *string `json:"myReaction"`
}

// ProjectCommentHTTP is one node in the comment tree.
type ProjectCommentHTTP struct {
	ID        string                `json:"id"`
	Body      string                `json:"body"`
	CreatedAt string                `json:"createdAt"`
	Author    CommentAuthorHTTP     `json:"author"`
	Reactions CommentReactionsHTTP  `json:"reactions"`
	Replies   []ProjectCommentHTTP  `json:"replies"`
}

// ProjectCommentsHTTP is the list response.
type ProjectCommentsHTTP struct {
	Comments []ProjectCommentHTTP `json:"comments"`
}

// CreateCommentRequest is the POST body.
type CreateCommentRequest struct {
	Body     string  `json:"body"`
	ParentID *string `json:"parentId"`
}
