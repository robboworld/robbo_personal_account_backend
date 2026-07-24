package usecase

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
	"go.uber.org/fx"
)

var ErrInvalidInput = errors.New("invalid notification input")
var ErrUserNotFound = errors.New("user not found")

type NotificationUseCase struct {
	gateway notifications.Gateway
}

type Module struct {
	fx.Out
	notifications.UseCase
}

func SetupNotificationUseCase(gateway notifications.Gateway) Module {
	return Module{UseCase: &NotificationUseCase{gateway: gateway}}
}

func (u *NotificationUseCase) Feed(userID string, limit int) ([]models.NotificationFeedItemHTTP, error) {
	if limit < 1 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	return u.gateway.ListFeed(strings.TrimSpace(userID), limit)
}

func (u *NotificationUseCase) UnreadCount(userID string) (int64, error) {
	return u.gateway.CountUnread(strings.TrimSpace(userID))
}

func (u *NotificationUseCase) MarkPersonalRead(userID, notificationID string) error {
	if strings.TrimSpace(notificationID) == "" {
		return ErrInvalidInput
	}
	return u.gateway.MarkPersonalRead(strings.TrimSpace(userID), notificationID)
}

func (u *NotificationUseCase) MarkAnnouncementRead(userID, announcementID string) error {
	if strings.TrimSpace(announcementID) == "" {
		return ErrInvalidInput
	}
	return u.gateway.MarkAnnouncementRead(strings.TrimSpace(userID), announcementID)
}

func (u *NotificationUseCase) MarkAllRead(userID string) error {
	return u.gateway.MarkAllRead(strings.TrimSpace(userID))
}

func (u *NotificationUseCase) SendPersonal(input notifications.PersonalInput) error {
	input.RecipientUserID = strings.TrimSpace(input.RecipientUserID)
	input.RecipientUsername = strings.TrimSpace(input.RecipientUsername)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" || input.Body == "" {
		return ErrInvalidInput
	}
	if input.RecipientUserID == "" {
		if input.RecipientUsername == "" {
			return ErrInvalidInput
		}
		id, err := usersearch.ResolveUsernameToID(input.RecipientUsername)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return ErrUserNotFound
			}
			return err
		}
		input.RecipientUserID = id
	}
	if input.Kind == "" {
		input.Kind = "admin_message"
	}
	if input.Severity == "" {
		input.Severity = "INFO"
	}
	return u.gateway.CreatePersonal(&models.UserNotificationDB{
		RecipientUserID: input.RecipientUserID,
		Title:           input.Title,
		Body:            input.Body,
		Kind:            input.Kind,
		Severity:        input.Severity,
		Source:          "admin",
		ActionURL:       input.ActionURL,
	})
}

func (u *NotificationUseCase) CreateAnnouncement(input notifications.AnnouncementInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" || input.Body == "" {
		return ErrInvalidInput
	}
	if input.AudienceJSON == "" {
		input.AudienceJSON = `{"all":true}`
	}
	var audience map[string]interface{}
	if err := json.Unmarshal([]byte(input.AudienceJSON), &audience); err != nil {
		return ErrInvalidInput
	}
	creator := strings.TrimSpace(input.CreatedByUserID)
	return u.gateway.CreateAnnouncement(&models.SystemAnnouncementDB{
		Title:           input.Title,
		Body:            input.Body,
		AudienceJSON:    input.AudienceJSON,
		CreatedByUserID: &creator,
		ExpiresAt:       input.ExpiresAt,
	})
}
