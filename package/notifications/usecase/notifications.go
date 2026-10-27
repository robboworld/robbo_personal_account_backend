package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users/profileaccess"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
	"go.uber.org/fx"
)

var ErrInvalidInput = errors.New("invalid notification input")
var ErrUserNotFound = errors.New("user not found")
var ErrNotYourStudent = errors.New("recipient is not a student of your classes")

const (
	maxTeacherRecipients = 200
	maxTeacherTitleRunes = 200
	maxTeacherBodyRunes  = 2000
	// teacherDailySends: sends per teacher in any 24 hours; a message to the whole class is one send.
	teacherDailySends  = 5
	teacherQuotaWindow = 24 * time.Hour
)

// teacherHasStudent reports whether studentID is in one of teacherID's active classes.
// The same rule decides whose profile a teacher may read; a variable so tests can stand in for LMS.
var teacherHasStudent = func(teacherID, studentID string) bool {
	reader := profileaccess.TeacherStudentReader
	return reader != nil && reader(teacherID, studentID)
}

// senderDisplayName is the teacher's name from the LMS profile, "" when unknown.
var senderDisplayName = licensing.LMSUserDisplayName

var now = time.Now

func teacherDailyLimit(role models.Role) int {
	if role == models.SuperAdmin {
		return 0
	}
	return teacherDailySends
}

func quotaOf(limit, used int) notifications.TeacherQuota {
	quota := notifications.TeacherQuota{Limit: limit, Used: used}
	if limit > 0 && used < limit {
		quota.Remaining = limit - used
	}
	return quota
}

func newSendID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type NotificationUseCase struct {
	gateway notifications.Gateway
	// lms: Open edX notifications of the user (nil = none); they join the feed and the counter.
	lms       notifications.LMSSource
	lmsCounts lmsCountCache
}

type Module struct {
	fx.Out
	notifications.UseCase
}

func SetupNotificationUseCase(gateway notifications.Gateway, lms notifications.LMSSource) Module {
	return Module{UseCase: &NotificationUseCase{gateway: gateway, lms: lms}}
}

// lmsCountTTL: every open LK tab asks for the counter every 30 s; the LMS is asked at most this often
// per user. Marking read here forgets the user's number at once.
const lmsCountTTL = 20 * time.Second

// lmsCountMaxUsers: past this many users the cache starts over instead of growing.
const lmsCountMaxUsers = 10000

type lmsCount struct {
	count int64
	at    time.Time
}

type lmsCountCache struct {
	mu     sync.Mutex
	byUser map[string]lmsCount
}

func (c *lmsCountCache) get(userID string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.byUser[userID]
	if !ok || now().Sub(entry.at) > lmsCountTTL {
		return 0, false
	}
	return entry.count, true
}

func (c *lmsCountCache) set(userID string, count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byUser == nil || len(c.byUser) >= lmsCountMaxUsers {
		c.byUser = map[string]lmsCount{}
	}
	c.byUser[userID] = lmsCount{count: count, at: now()}
}

func (c *lmsCountCache) forget(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.byUser, userID)
}

func (u *NotificationUseCase) lmsOn() bool {
	return u.lms != nil && u.lms.Enabled()
}

// Feed: LK notifications and announcements and/or the user's Open edX notifications, newest first.
// An LMS that fails or is slow leaves its part out: the LK part is still shown.
func (u *NotificationUseCase) Feed(userID string, limit int, source string) (models.NotificationFeedHTTP, error) {
	if limit < 1 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if source != notifications.SourceAll && source != notifications.SourceLK && source != notifications.SourceLMS {
		return models.NotificationFeedHTTP{}, ErrInvalidInput
	}
	userID = strings.TrimSpace(userID)
	var lmsFeed notifications.LMSFeed
	var lmsErr error
	var wg sync.WaitGroup
	if u.lmsOn() && source != notifications.SourceLK {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lmsFeed, lmsErr = u.lms.Feed(userID, limit)
		}()
	}
	var own []models.NotificationFeedItemHTTP
	var err error
	if source != notifications.SourceLMS {
		own, err = u.gateway.ListFeed(userID, limit)
	}
	wg.Wait()
	if err != nil {
		return models.NotificationFeedHTTP{}, err
	}
	if lmsErr != nil {
		log.Printf("[notifications] LMS feed of user %s left out: %v", userID, lmsErr)
	}
	items := append(own, lmsFeed.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	if items == nil {
		items = []models.NotificationFeedItemHTTP{}
	}
	return models.NotificationFeedHTTP{Items: items, LMS: lmsFeed.Links}, nil
}

// UnreadCount: unread LK notifications and unread Open edX ones (cached for lmsCountTTL).
func (u *NotificationUseCase) UnreadCount(userID string) (notifications.UnreadCounts, error) {
	userID = strings.TrimSpace(userID)
	lk, err := u.gateway.CountUnread(userID)
	if err != nil {
		return notifications.UnreadCounts{}, err
	}
	counts := notifications.UnreadCounts{Count: lk, LK: lk, LMSEnabled: u.lmsOn()}
	if !counts.LMSEnabled {
		return counts, nil
	}
	lmsUnread, ok := u.lmsCounts.get(userID)
	if !ok {
		lmsUnread, err = u.lms.UnreadCount(userID)
		if err != nil {
			log.Printf("[notifications] LMS unread count of user %s left out: %v", userID, err)
			return counts, nil
		}
		u.lmsCounts.set(userID, lmsUnread)
	}
	counts.LMS = lmsUnread
	counts.Count += lmsUnread
	return counts, nil
}

// MarkLMSRead: an Open edX notification read in LK is read in the LMS too (one store).
func (u *NotificationUseCase) MarkLMSRead(userID, notificationID string) error {
	id, err := strconv.ParseInt(strings.TrimSpace(notificationID), 10, 64)
	if err != nil || id <= 0 || !u.lmsOn() {
		return ErrInvalidInput
	}
	userID = strings.TrimSpace(userID)
	u.lmsCounts.forget(userID)
	return u.lms.MarkRead(userID, []int64{id})
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

// MarkAllRead: LK notifications and announcements and/or the Open edX ones. With both, an LMS
// failure is logged: the LK part is done and the LMS items stay unread until the next try.
func (u *NotificationUseCase) MarkAllRead(userID string, source string) error {
	userID = strings.TrimSpace(userID)
	switch source {
	case notifications.SourceAll, notifications.SourceLK, notifications.SourceLMS:
	default:
		return ErrInvalidInput
	}
	if source != notifications.SourceLMS {
		if err := u.gateway.MarkAllRead(userID); err != nil {
			return err
		}
	}
	if source == notifications.SourceLK || !u.lmsOn() {
		return nil
	}
	u.lmsCounts.forget(userID)
	err := u.lms.MarkAllRead(userID)
	if err != nil && source == notifications.SourceAll {
		log.Printf("[notifications] LMS mark-all-read of user %s failed: %v", userID, err)
		return nil
	}
	return err
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

// SendFromTeacher sends one message to each recipient. A teacher may write only to students of
// their own classes; one stranger in the list rejects the whole send. A teacher has
// teacherDailySends sends per 24 hours; SuperAdmin writes to anyone without a limit.
// The title is prefixed with the sender's name, so a student with several teachers knows who wrote.
func (u *NotificationUseCase) SendFromTeacher(input notifications.TeacherMessageInput) (int, notifications.TeacherQuota, error) {
	limit := teacherDailyLimit(input.SenderRole)
	sender := strings.TrimSpace(input.SenderUserID)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	if sender == "" || title == "" || body == "" ||
		utf8.RuneCountInString(title) > maxTeacherTitleRunes ||
		utf8.RuneCountInString(body) > maxTeacherBodyRunes {
		return 0, notifications.TeacherQuota{}, ErrInvalidInput
	}
	seen := make(map[string]bool, len(input.RecipientUserIDs))
	recipients := make([]string, 0, len(input.RecipientUserIDs))
	for _, id := range input.RecipientUserIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		recipients = append(recipients, id)
	}
	if len(recipients) == 0 || len(recipients) > maxTeacherRecipients {
		return 0, notifications.TeacherQuota{}, ErrInvalidInput
	}
	if input.SenderRole != models.SuperAdmin {
		for _, id := range recipients {
			if !teacherHasStudent(sender, id) {
				return 0, notifications.TeacherQuota{}, ErrNotYourStudent
			}
		}
	}
	if name := strings.TrimSpace(senderDisplayName(sender)); name != "" {
		title = name + ": " + title
	}
	sendID, err := newSendID()
	if err != nil {
		return 0, notifications.TeacherQuota{}, err
	}
	rows := make([]models.UserNotificationDB, 0, len(recipients))
	for _, id := range recipients {
		rows = append(rows, models.UserNotificationDB{
			RecipientUserID: id,
			Title:           title,
			Body:            body,
			Kind:            "teacher_message",
			Severity:        "INFO",
			Source:          "teacher",
		})
	}
	used, err := u.gateway.CreateTeacherSend(sender, sendID, rows, limit, now().Add(-teacherQuotaWindow))
	if err != nil {
		return 0, quotaOf(limit, used), err
	}
	return len(rows), quotaOf(limit, used+1), nil
}

func (u *NotificationUseCase) TeacherQuota(senderUserID string, senderRole models.Role) (notifications.TeacherQuota, error) {
	limit := teacherDailyLimit(senderRole)
	if limit == 0 {
		return quotaOf(0, 0), nil
	}
	used, err := u.gateway.CountTeacherSends(strings.TrimSpace(senderUserID), now().Add(-teacherQuotaWindow))
	if err != nil {
		return notifications.TeacherQuota{}, err
	}
	return quotaOf(limit, used), nil
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
