package models

import "time"

type UserNotificationDB struct {
	ID              string     `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RecipientUserID string     `gorm:"column:recipient_user_id"`
	Title           string     `gorm:"column:title"`
	Body            string     `gorm:"column:body"`
	Kind            string     `gorm:"column:kind"`
	Severity        string     `gorm:"column:severity"`
	Source          string     `gorm:"column:source"`
	ActionURL       *string    `gorm:"column:action_url"`
	DedupeKey       *string    `gorm:"column:dedupe_key"`
	ReadAt          *time.Time `gorm:"column:read_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (UserNotificationDB) TableName() string { return "user_notifications" }

type SystemAnnouncementDB struct {
	ID              string     `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Title           string     `gorm:"column:title"`
	Body            string     `gorm:"column:body"`
	AudienceJSON    string     `gorm:"column:audience_json"`
	CreatedByUserID *string    `gorm:"column:created_by_user_id"`
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (SystemAnnouncementDB) TableName() string { return "system_announcements" }

type AnnouncementReadDB struct {
	AnnouncementID string    `gorm:"column:announcement_id;primaryKey"`
	UserID         string    `gorm:"column:user_id;primaryKey"`
	ReadAt         time.Time `gorm:"column:read_at"`
}

func (AnnouncementReadDB) TableName() string { return "announcement_reads" }

type NotificationFeedItemHTTP struct {
	FeedKind  string     `json:"feedKind"`
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Kind      string     `json:"kind"`
	Severity  string     `json:"severity"`
	Source    string     `json:"source"`
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
	ActionURL *string    `json:"actionUrl"`
}
