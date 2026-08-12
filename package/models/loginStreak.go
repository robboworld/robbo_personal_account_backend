package models

import "time"

// UserLoginStreakDB maps user_login_streaks (Projects Postgres).
type UserLoginStreakDB struct {
	LmsUserID     string     `gorm:"column:lms_user_id;primaryKey"`
	CurrentStreak int        `gorm:"column:current_streak"`
	LongestStreak int        `gorm:"column:longest_streak"`
	LastLoginDate *time.Time `gorm:"column:last_login_date;type:date"`
	Timezone      string     `gorm:"column:timezone"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (UserLoginStreakDB) TableName() string { return "user_login_streaks" }

// LoginStreakHTTP is returned from check-auth and GraphQL.
type LoginStreakHTTP struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}
