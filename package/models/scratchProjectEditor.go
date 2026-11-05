package models

import "time"

// EditingTTLSeconds is how long a signal from the editor counts: it comes every 30 seconds, and a
// background tab may be slowed down to once a minute.
const EditingTTLSeconds = 90

// Who is editing handed-in work.
const (
	EditorTeacher = "teacher"
	EditorStudent = "student"
)

// ScratchProjectEditorDB maps scratch_project_editors: one person editing one piece of work.
type ScratchProjectEditorDB struct {
	ProjectID  string    `gorm:"column:project_id;primaryKey"`
	UserID     string    `gorm:"column:user_id;primaryKey"`
	Role       string    `gorm:"column:role"`
	LastSeenAt time.Time `gorm:"column:last_seen_at"`
}

func (ScratchProjectEditorDB) TableName() string { return "scratch_project_editors" }

// ProjectEditor is someone editing the work as others see it.
type ProjectEditor struct {
	Role string `json:"role"`
	Name string `json:"name,omitempty"`
}
