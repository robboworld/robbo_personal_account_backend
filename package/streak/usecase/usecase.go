package usecase

import (
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/streak"
	"go.uber.org/fx"
)

type StreakUseCaseImpl struct {
	gateway streak.Gateway
	now     func() time.Time
}

type StreakUseCaseModule struct {
	fx.Out
	streak.UseCase
}

func SetupStreakUseCase(gateway streak.Gateway) StreakUseCaseModule {
	return StreakUseCaseModule{
		UseCase: &StreakUseCaseImpl{
			gateway: gateway,
			now:     time.Now,
		},
	}
}

// NewStreakUseCaseForTest builds a usecase with a fixed clock (unit tests).
func NewStreakUseCaseForTest(gateway streak.Gateway, now func() time.Time) streak.UseCase {
	return &StreakUseCaseImpl{gateway: gateway, now: now}
}

func (u *StreakUseCaseImpl) Get(lmsUserID string) (*models.LoginStreakHTTP, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return &models.LoginStreakHTTP{}, nil
	}
	row, err := u.gateway.GetByUserID(lmsUserID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &models.LoginStreakHTTP{}, nil
	}
	return &models.LoginStreakHTTP{
		Current: row.CurrentStreak,
		Longest: row.LongestStreak,
	}, nil
}

func (u *StreakUseCaseImpl) RecordVisit(lmsUserID, timezone string) (*models.LoginStreakHTTP, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return &models.LoginStreakHTTP{}, nil
	}
	loc := loadLocation(timezone)
	now := u.now().In(loc)
	today := dateOnly(now)

	row, err := u.gateway.GetByUserID(lmsUserID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &models.UserLoginStreakDB{LmsUserID: lmsUserID}
	}

	tzName := timezone
	if strings.TrimSpace(tzName) == "" {
		tzName = "UTC"
	}

	if row.LastLoginDate != nil {
		last := dateOnly(row.LastLoginDate.In(time.UTC))
		if last.Equal(today) {
			row.Timezone = tzName
			row.UpdatedAt = u.now().UTC()
			if err := u.gateway.Upsert(row); err != nil {
				return nil, err
			}
			return &models.LoginStreakHTTP{Current: row.CurrentStreak, Longest: row.LongestStreak}, nil
		}
		yesterday := today.AddDate(0, 0, -1)
		if last.Equal(yesterday) {
			row.CurrentStreak++
		} else {
			row.CurrentStreak = 1
		}
	} else {
		row.CurrentStreak = 1
	}

	if row.CurrentStreak > row.LongestStreak {
		row.LongestStreak = row.CurrentStreak
	}
	row.LastLoginDate = &today
	row.Timezone = tzName
	row.UpdatedAt = u.now().UTC()

	if err := u.gateway.Upsert(row); err != nil {
		return nil, err
	}
	return &models.LoginStreakHTTP{Current: row.CurrentStreak, Longest: row.LongestStreak}, nil
}

func (u *StreakUseCaseImpl) Increment(lmsUserID string) (*models.LoginStreakHTTP, error) {
	lmsUserID = strings.TrimSpace(lmsUserID)
	if lmsUserID == "" {
		return &models.LoginStreakHTTP{}, nil
	}
	row, err := u.gateway.GetByUserID(lmsUserID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &models.UserLoginStreakDB{LmsUserID: lmsUserID}
	}
	row.CurrentStreak++
	if row.CurrentStreak > row.LongestStreak {
		row.LongestStreak = row.CurrentStreak
	}
	today := dateOnly(u.now().UTC())
	row.LastLoginDate = &today
	row.UpdatedAt = u.now().UTC()
	if err := u.gateway.Upsert(row); err != nil {
		return nil, err
	}
	return &models.LoginStreakHTTP{Current: row.CurrentStreak, Longest: row.LongestStreak}, nil
}

func loadLocation(tz string) *time.Location {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
