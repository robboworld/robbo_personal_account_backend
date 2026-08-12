package usecase

import (
	"sync"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type memGateway struct {
	mu   sync.Mutex
	rows map[string]*models.UserLoginStreakDB
}

func newMemGateway() *memGateway {
	return &memGateway{rows: map[string]*models.UserLoginStreakDB{}}
}

func (g *memGateway) GetByUserID(lmsUserID string) (*models.UserLoginStreakDB, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	row, ok := g.rows[lmsUserID]
	if !ok {
		return nil, nil
	}
	cp := *row
	if row.LastLoginDate != nil {
		t := *row.LastLoginDate
		cp.LastLoginDate = &t
	}
	return &cp, nil
}

func (g *memGateway) Upsert(row *models.UserLoginStreakDB) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	cp := *row
	if row.LastLoginDate != nil {
		t := *row.LastLoginDate
		cp.LastLoginDate = &t
	}
	g.rows[row.LmsUserID] = &cp
	return nil
}

func TestRecordVisit_FirstDay(t *testing.T) {
	gw := newMemGateway()
	now := time.Date(2026, 7, 31, 15, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })

	got, err := uc.RecordVisit("42", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 || got.Longest != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestRecordVisit_SameDayIdempotent(t *testing.T) {
	gw := newMemGateway()
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })

	if _, err := uc.RecordVisit("42", "UTC"); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 7, 31, 22, 0, 0, 0, time.UTC)
	got, err := uc.RecordVisit("42", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 {
		t.Fatalf("expected still 1, got %d", got.Current)
	}
}

func TestRecordVisit_ConsecutiveDay(t *testing.T) {
	gw := newMemGateway()
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })
	if _, err := uc.RecordVisit("42", "UTC"); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	got, err := uc.RecordVisit("42", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 2 || got.Longest != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestRecordVisit_GapResets(t *testing.T) {
	gw := newMemGateway()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })
	if _, err := uc.RecordVisit("42", "UTC"); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	got, err := uc.RecordVisit("42", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 {
		t.Fatalf("expected reset to 1, got %d", got.Current)
	}
	if got.Longest != 1 {
		t.Fatalf("longest should stay 1, got %d", got.Longest)
	}
}

func TestRecordVisit_InvalidTZFallsBackUTC(t *testing.T) {
	gw := newMemGateway()
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })
	got, err := uc.RecordVisit("42", "Not/AZone")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestRecordVisit_MoscowTimezoneBoundary(t *testing.T) {
	gw := newMemGateway()
	// 2026-07-30 22:00 UTC == 2026-07-31 01:00 MSK
	now := time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)
	uc := NewStreakUseCaseForTest(gw, func() time.Time { return now })
	got, err := uc.RecordVisit("42", "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 {
		t.Fatalf("got %+v", got)
	}
	row, _ := gw.GetByUserID("42")
	if row.LastLoginDate == nil {
		t.Fatal("missing last_login_date")
	}
	y, m, d := row.LastLoginDate.Date()
	if y != 2026 || m != time.July || d != 31 {
		t.Fatalf("expected MSK calendar day 31, got %v", row.LastLoginDate)
	}
}
