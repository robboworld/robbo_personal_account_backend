package models

import (
	"time"

	"github.com/lib/pq"
)

// ClassKidLoginRoomDB is the class sign-in page of a ЛК class (architecture/ADR_CLASS_KID_LOGIN.md).
// LoginSlug is apart from the invite slug: the page lists the children's names. OpenUntil is the
// end of the lesson window for picture sign-in; nil or past means closed. RoomCode is the six
// digits shown on the projector during the lesson: /k takes them instead of the page address.
type ClassKidLoginRoomDB struct {
	ClassID   string     `gorm:"type:uuid;primaryKey;column:class_id"`
	LoginSlug string     `gorm:"column:login_slug"`
	OpenUntil *time.Time `gorm:"column:open_until"`
	RoomCode  *string    `gorm:"column:room_code"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (ClassKidLoginRoomDB) TableName() string { return "class_kid_login_rooms" }

// IsOpen: picture sign-in is open at now.
func (r *ClassKidLoginRoomDB) IsOpen(now time.Time) bool {
	return r != nil && r.OpenUntil != nil && r.OpenUntil.After(now)
}

// ClassKidLoginDB is a kid account a teacher made for their class. UserID is its LMS user;
// Pictures are the KidSecretLength picture numbers (0..KidPictureCount-1) to tap, in order.
type ClassKidLoginDB struct {
	UserID         string        `gorm:"primaryKey;column:user_id"`
	ClassID        string        `gorm:"type:uuid;column:class_id"`
	DisplayName    string        `gorm:"column:display_name"`
	Avatar         int           `gorm:"column:avatar"`
	Pictures       pq.Int64Array `gorm:"column:pictures;type:smallint[]"`
	CardTokenHash  *string       `gorm:"column:card_token_hash"`
	FailedAttempts int           `gorm:"column:failed_attempts"`
	LockedAt       *time.Time    `gorm:"column:locked_at"`
	CreatedBy      string        `gorm:"column:created_by"`
	CreatedAt      time.Time     `gorm:"column:created_at"`
	LastLoginAt    *time.Time    `gorm:"column:last_login_at"`
}

func (ClassKidLoginDB) TableName() string { return "class_kid_logins" }

const (
	// KidPictureCount pictures on the sign-in grid.
	KidPictureCount = 9
	// KidSecretLength pictures make a secret, in order, repeats allowed: 9^5 = 59049.
	KidSecretLength = 5
	// KidAvatarCount animals a class hands out, one per child.
	KidAvatarCount = 30
)

// Class history of kid accounts. Detail by action:
//   - HistoryKidsCreated: how many were created; Title is "consent" (the teacher confirmed
//     the parents' consent);
//   - HistoryKidLoginOpened: the end of the window, RFC 3339;
//   - HistoryKidLogin: "card" for a QR card, empty for pictures.
//
// TargetUserID is the child for the per-child actions.
const (
	HistoryKidsCreated     = "kids_created"
	HistoryKidLoginOpened  = "kid_login_opened"
	HistoryKidLoginClosed  = "kid_login_closed"
	HistoryKidPicturesSet  = "kid_pictures_reset"
	HistoryKidCardsIssued  = "kid_cards_issued"
	HistoryKidUnlocked     = "kid_unlocked"
	HistoryKidReturned     = "kid_returned"
	HistoryKidLogin        = "kid_login"
	HistoryKidLoginCardTag = "card"
)
