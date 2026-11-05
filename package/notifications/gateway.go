package notifications

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type Gateway interface {
	CreateOrUpdateByDedupe(notification *models.UserNotificationDB) error
	// AddToDigest keeps one row per DedupeKey for repeated events: the title is titlePrefix and
	// their count, the body is bodyPrefix and the items. The caller decides what counts as an event.
	AddToDigest(notification *models.UserNotificationDB, titlePrefix, bodyPrefix, item string) error
	CreatePersonal(notification *models.UserNotificationDB) error
	// CreateTeacherSend writes one send of senderID unless they already made dailyLimit sends
	// since `since` (0 = no limit); returns the sends made before this one.
	CreateTeacherSend(senderID, sendID string, rows []models.UserNotificationDB, dailyLimit int, since time.Time) (usedBefore int, err error)
	CountTeacherSends(senderID string, since time.Time) (int, error)
	ListFeed(userID string, limit int) ([]models.NotificationFeedItemHTTP, error)
	CountUnread(userID string) (int64, error)
	MarkPersonalRead(userID, notificationID string) error
	MarkAnnouncementRead(userID, announcementID string) error
	MarkAllRead(userID string) error
	CreateAnnouncement(announcement *models.SystemAnnouncementDB) error
}
