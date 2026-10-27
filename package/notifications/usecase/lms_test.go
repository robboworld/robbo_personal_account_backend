package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
)

type feedGateway struct {
	notifications.Gateway
	items     []models.NotificationFeedItemHTTP
	unread    int64
	markedAll int
}

func (f *feedGateway) ListFeed(string, int) ([]models.NotificationFeedItemHTTP, error) {
	return append([]models.NotificationFeedItemHTTP(nil), f.items...), nil
}

func (f *feedGateway) CountUnread(string) (int64, error) { return f.unread, nil }

func (f *feedGateway) MarkAllRead(string) error {
	f.markedAll++
	return nil
}

type fakeLMS struct {
	feed      notifications.LMSFeed
	err       error
	unread    int64
	counts    int
	read      [][]int64
	markedAll int
}

func (f *fakeLMS) Enabled() bool { return true }

func (f *fakeLMS) Feed(string, int) (notifications.LMSFeed, error) { return f.feed, f.err }

func (f *fakeLMS) UnreadCount(string) (int64, error) {
	f.counts++
	return f.unread, f.err
}

func (f *fakeLMS) MarkRead(_ string, ids []int64) error {
	f.read = append(f.read, ids)
	return f.err
}

func (f *fakeLMS) MarkAllRead(string) error {
	f.markedAll++
	return f.err
}

func at(hour int) time.Time { return time.Date(2026, 10, 27, hour, 0, 0, 0, time.UTC) }

func TestFeedMergesLMSNewestFirst(t *testing.T) {
	gw := &feedGateway{items: []models.NotificationFeedItemHTTP{
		{FeedKind: "personal", ID: "a", CreatedAt: at(12)},
		{FeedKind: "announcement", ID: "b", CreatedAt: at(8)},
	}}
	links := &models.LMSNotificationLinksHTTP{Feed: "f", Settings: "s"}
	lms := &fakeLMS{feed: notifications.LMSFeed{Enabled: true, Links: links, Items: []models.NotificationFeedItemHTTP{
		{FeedKind: "lms", ID: "7", CreatedAt: at(10)},
		{FeedKind: "lms", ID: "6", CreatedAt: at(6)},
	}}}
	uc := &NotificationUseCase{gateway: gw, lms: lms}
	ids := func(feed models.NotificationFeedHTTP) string {
		got := ""
		for _, item := range feed.Items {
			got += item.ID
		}
		return got
	}
	feed, err := uc.Feed("789", 3, notifications.SourceAll)
	if err != nil || ids(feed) != "a7b" || feed.LMS != links {
		t.Fatalf("all: feed = %q, links = %v, err = %v", ids(feed), feed.LMS, err)
	}
	// The bell's tabs: LK alone is not crowded out by newer LMS items, LMS alone keeps its links.
	lk, err := uc.Feed("789", 3, notifications.SourceLK)
	if err != nil || ids(lk) != "ab" || lk.LMS != nil {
		t.Fatalf("lk: feed = %q, links = %v, err = %v", ids(lk), lk.LMS, err)
	}
	lmsOnly, err := uc.Feed("789", 3, notifications.SourceLMS)
	if err != nil || ids(lmsOnly) != "76" || lmsOnly.LMS != links {
		t.Fatalf("lms: feed = %q, links = %v, err = %v", ids(lmsOnly), lmsOnly.LMS, err)
	}
	if _, err := uc.Feed("789", 3, "other"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown source: %v", err)
	}
}

func TestFeedWithoutLMSWhenItFails(t *testing.T) {
	gw := &feedGateway{items: []models.NotificationFeedItemHTTP{{FeedKind: "personal", ID: "a", CreatedAt: at(12)}}}
	uc := &NotificationUseCase{gateway: gw, lms: &fakeLMS{err: errors.New("down")}}
	feed, err := uc.Feed("789", 30, notifications.SourceAll)
	if err != nil || len(feed.Items) != 1 || feed.LMS != nil {
		t.Fatalf("feed = %+v, err = %v", feed, err)
	}
	empty, err := (&NotificationUseCase{gateway: &feedGateway{}}).Feed("789", 30, notifications.SourceAll)
	if err != nil || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("no LMS, no items: %+v, %v (items must be [] for the JSON)", empty, err)
	}
}

func TestUnreadCountAddsCachedLMS(t *testing.T) {
	gw := &feedGateway{unread: 2}
	lms := &fakeLMS{unread: 3}
	uc := &NotificationUseCase{gateway: gw, lms: lms}
	for i := 0; i < 2; i++ {
		counts, err := uc.UnreadCount("789")
		if err != nil || counts != (notifications.UnreadCounts{Count: 5, LK: 2, LMS: 3, LMSEnabled: true}) {
			t.Fatalf("counts = %+v, err = %v", counts, err)
		}
	}
	if lms.counts != 1 {
		t.Fatalf("LMS asked %d times, want 1 (cached)", lms.counts)
	}
	if err := uc.MarkLMSRead("789", "7"); err != nil {
		t.Fatal(err)
	}
	lms.unread = 2
	if counts, _ := uc.UnreadCount("789"); counts.Count != 4 || lms.counts != 2 {
		t.Fatalf("after read: counts = %+v, LMS asked %d times (the cache must be forgotten)", counts, lms.counts)
	}
	lms.err = errors.New("down")
	uc.lmsCounts.forget("789")
	if counts, err := uc.UnreadCount("789"); err != nil || counts != (notifications.UnreadCounts{Count: 2, LK: 2, LMSEnabled: true}) {
		t.Fatalf("LMS down: counts = %+v, err = %v (LK part only)", counts, err)
	}
}

func TestMarkLMSRead(t *testing.T) {
	lms := &fakeLMS{}
	uc := &NotificationUseCase{gateway: &feedGateway{}, lms: lms}
	for _, bad := range []string{"", "x", "0", "-1"} {
		if err := uc.MarkLMSRead("789", bad); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("MarkLMSRead(%q) = %v", bad, err)
		}
	}
	if err := uc.MarkLMSRead("789", "7"); err != nil || len(lms.read) != 1 || lms.read[0][0] != 7 {
		t.Fatalf("err = %v, read = %v", err, lms.read)
	}
	if err := (&NotificationUseCase{gateway: &feedGateway{}}).MarkLMSRead("789", "7"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("no LMS source: %v", err)
	}
}

func TestMarkAllReadIncludesLMS(t *testing.T) {
	gw := &feedGateway{}
	lms := &fakeLMS{}
	uc := &NotificationUseCase{gateway: gw, lms: lms}
	if err := uc.MarkAllRead("789", notifications.SourceAll); err != nil || gw.markedAll != 1 || lms.markedAll != 1 {
		t.Fatalf("all: err = %v, LK = %d, LMS = %d", err, gw.markedAll, lms.markedAll)
	}
	if err := uc.MarkAllRead("789", notifications.SourceLK); err != nil || gw.markedAll != 2 || lms.markedAll != 1 {
		t.Fatalf("lk: err = %v, LK = %d, LMS = %d", err, gw.markedAll, lms.markedAll)
	}
	if err := uc.MarkAllRead("789", notifications.SourceLMS); err != nil || gw.markedAll != 2 || lms.markedAll != 2 {
		t.Fatalf("lms: err = %v, LK = %d, LMS = %d", err, gw.markedAll, lms.markedAll)
	}
	lms.err = errors.New("down")
	if err := uc.MarkAllRead("789", notifications.SourceAll); err != nil {
		t.Fatalf("an LMS failure must not fail mark-all-read of both: %v", err)
	}
	if err := uc.MarkAllRead("789", notifications.SourceLMS); err == nil {
		t.Fatal("the LMS tab must report an LMS failure")
	}
	if err := uc.MarkAllRead("789", "other"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown source: %v", err)
	}
}
