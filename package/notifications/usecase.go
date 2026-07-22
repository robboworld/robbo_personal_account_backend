package notifications

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type PersonalInput struct {
	RecipientUserID string
	Title           string
	Body            string
	Kind            string
	Severity        string
	ActionURL       *string
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
	CreateAnnouncement(input AnnouncementInput) error
}
