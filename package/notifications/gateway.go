package notifications

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

type Gateway interface {
	CreateOrUpdateByDedupe(notification *models.UserNotificationDB) error
	CreatePersonal(notification *models.UserNotificationDB) error
	ListFeed(userID string, limit int) ([]models.NotificationFeedItemHTTP, error)
	CountUnread(userID string) (int64, error)
	MarkPersonalRead(userID, notificationID string) error
	MarkAnnouncementRead(userID, announcementID string) error
	MarkAllRead(userID string) error
	CreateAnnouncement(announcement *models.SystemAnnouncementDB) error
}
