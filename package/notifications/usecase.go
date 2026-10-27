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

// Feed sources: the bell shows LK notifications first and Open edX ones as a secondary tab.
const (
	SourceAll = ""
	SourceLK  = "lk"
	SourceLMS = "lms"
)

// UnreadCounts: unread LK notifications (with announcements) and unread Open edX ones;
// LMSEnabled says whether the bell has the Open edX tab at all.
type UnreadCounts struct {
	Count      int64 `json:"count"`
	LK         int64 `json:"lk"`
	LMS        int64 `json:"lms"`
	LMSEnabled bool  `json:"lmsEnabled"`
}

type UseCase interface {
	// Feed: source "lk" (LK notifications and announcements), "lms" (Open edX ones) or "" (both).
	Feed(userID string, limit int, source string) (models.NotificationFeedHTTP, error)
	UnreadCount(userID string) (UnreadCounts, error)
	MarkPersonalRead(userID, notificationID string) error
	MarkAnnouncementRead(userID, announcementID string) error
	MarkLMSRead(userID, notificationID string) error
	// MarkAllRead: source "lk", "lms" or "" (both).
	MarkAllRead(userID string, source string) error
	SendPersonal(input PersonalInput) error
	SendFromTeacher(input TeacherMessageInput) (sent int, quota TeacherQuota, err error)
	TeacherQuota(senderUserID string, senderRole models.Role) (TeacherQuota, error)
	CreateAnnouncement(input AnnouncementInput) error
}
