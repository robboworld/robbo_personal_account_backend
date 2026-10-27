package notifications

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

// LMSSource is the user's Open edX notifications, read through the LMS service API
// (gateway/lms.go). The LMS keeps them and their «прочитано»: LK only shows them in its bell,
// so marking one read here is the same as in the LMS bell and back.
type LMSSource interface {
	Enabled() bool
	Feed(userID string, limit int) (LMSFeed, error)
	UnreadCount(userID string) (int64, error)
	MarkRead(userID string, ids []int64) error
	MarkAllRead(userID string) error
}

// LMSFeed: items with feedKind "lms"; Enabled false when the user has no LMS notifications at all.
type LMSFeed struct {
	Enabled bool
	Items   []models.NotificationFeedItemHTTP
	Links   *models.LMSNotificationLinksHTTP
}
