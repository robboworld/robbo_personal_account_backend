package notifications

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type PersonalInput struct {
	RecipientUserID   string
	RecipientUsername string
	Title             string
	Body              string
	Kind              string
	Severity          string
	ActionURL         *string
}

// ErrTeacherDailyLimit: the teacher has used up today's sends; nothing was written.
var ErrTeacherDailyLimit = errors.New("teacher_daily_limit")

// TeacherQuota is how many sends a teacher has left in the last 24 hours. Limit 0 means unlimited.
type TeacherQuota struct {
	Limit     int `json:"limit"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

// TeacherMessageInput is a message from a teacher to students of their own classes.
type TeacherMessageInput struct {
	SenderUserID     string
	SenderRole       models.Role
	RecipientUserIDs []string
	Title            string
	Body             string
}

type AnnouncementInput struct {
	Title           string
	Body            string
	AudienceJSON    string
	CreatedByUserID string
	ExpiresAt       *time.Time
}

type UseCase interface {
	Feed(userID string, limit int) ([]models.NotificationFeedItemHTTP, error)
	UnreadCount(userID string) (int64, error)
	MarkPersonalRead(userID, notificationID string) error
	MarkAnnouncementRead(userID, announcementID string) error
	MarkAllRead(userID string) error
	SendPersonal(input PersonalInput) error
	SendFromTeacher(input TeacherMessageInput) (sent int, quota TeacherQuota, err error)
	TeacherQuota(senderUserID string, senderRole models.Role) (TeacherQuota, error)
	CreateAnnouncement(input AnnouncementInput) error
}
