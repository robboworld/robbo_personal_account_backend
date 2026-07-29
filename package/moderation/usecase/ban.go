package usecase

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var (
	ErrCannotBanSelf     = errors.New("cannot ban yourself")
	ErrAlreadyBanned     = errors.New("user is already banned")
	ErrNotBanned         = errors.New("user is not banned")
	ErrReasonRequired    = errors.New("ban reason is required")
	ErrInvalidExpiresAt  = errors.New("expiresAt must be in the future")
	ErrExpiresAtTooFar   = errors.New("expiresAt exceeds maximum ban duration")
)

const defaultMaxBanDays = 365

// BanUseCaseImpl implements moderation.UseCase.
type BanUseCaseImpl struct {
	bans     moderation.Gateway
	sessions licensing.Gateway
	search   *usersearch.Service
}

// BanUseCaseModule is the FX output.
type BanUseCaseModule struct {
	fx.Out
	moderation.UseCase
}

// SetupBanUseCase wires ban audit + LMS write + session revoke + ES badge sync.
func SetupBanUseCase(bans moderation.Gateway, sessions licensing.Gateway, search *usersearch.Service) BanUseCaseModule {
	return BanUseCaseModule{
		UseCase: &BanUseCaseImpl{bans: bans, sessions: sessions, search: search},
	}
}

func (u *BanUseCaseImpl) syncSearchActive(lmsUserID string) {
	if u.search == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := u.search.IndexUserByID(ctx, lmsUserID); err != nil {
		log.Printf("moderation: es index user %s: %v", lmsUserID, err)
	}
}

func maxBanDuration() time.Duration {
	days := viper.GetInt("moderation.maxBanDays")
	if days < 1 {
		days = defaultMaxBanDays
	}
	return time.Duration(days) * 24 * time.Hour
}

func parseLmsUserID(id string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil || n <= 0 {
		return 0, auth.ErrUserNotFound
	}
	return n, nil
}

func (u *BanUseCaseImpl) BanUser(callerID, targetID, reason string, expiresAt *time.Time) (*models.UserBanCore, error) {
	callerID = strings.TrimSpace(callerID)
	targetID = strings.TrimSpace(targetID)
	reason = strings.TrimSpace(reason)
	if callerID == "" || targetID == "" {
		return nil, auth.ErrUserNotFound
	}
	if callerID == targetID {
		return nil, ErrCannotBanSelf
	}
	if reason == "" {
		return nil, ErrReasonRequired
	}
	now := time.Now().UTC()
	if expiresAt != nil {
		exp := expiresAt.UTC()
		expiresAt = &exp
		if !expiresAt.After(now) {
			return nil, ErrInvalidExpiresAt
		}
		if expiresAt.After(now.Add(maxBanDuration())) {
			return nil, ErrExpiresAtTooFar
		}
	}

	targetNum, err := parseLmsUserID(targetID)
	if err != nil {
		return nil, err
	}

	if active, err := u.bans.GetActiveBan(targetID); err == nil && active != nil {
		return nil, ErrAlreadyBanned
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	if err := writer.SetUserActive(targetNum, false); err != nil {
		return nil, err
	}

	revoked := 0
	if u.sessions != nil {
		n, revErr := u.sessions.RevokeAllSessionsForUser(targetID)
		if revErr != nil {
			log.Printf("moderation ban: revoke sessions for %s: %v", targetID, revErr)
		} else {
			revoked = n
		}
	}

	ban, err := u.bans.CreateBan(&models.UserBanCore{
		LmsUserID:         targetID,
		Reason:            reason,
		BannedByLmsUserID: callerID,
		ExpiresAt:         expiresAt,
		RevokedSessions:   revoked,
	})
	if err != nil {
		// Best-effort rollback of LMS flag if audit insert fails.
		_ = writer.SetUserActive(targetNum, true)
		u.syncSearchActive(targetID)
		return nil, err
	}
	u.syncSearchActive(targetID)
	return ban, nil
}

func (u *BanUseCaseImpl) UnbanUser(callerID, targetID, reason string) (*models.UserBanCore, error) {
	callerID = strings.TrimSpace(callerID)
	targetID = strings.TrimSpace(targetID)
	reason = strings.TrimSpace(reason)
	if targetID == "" {
		return nil, auth.ErrUserNotFound
	}
	targetNum, err := parseLmsUserID(targetID)
	if err != nil {
		return nil, err
	}

	active, err := u.bans.GetOpenBan(targetID)
	if errors.Is(err, gorm.ErrRecordNotFound) || active == nil {
		// Still ensure LMS is active (repair path for manual is_active=0).
		writer, wErr := lmsdb.NewWriterFromConfig()
		if wErr != nil {
			return nil, ErrNotBanned
		}
		defer writer.Close()
		_ = writer.SetUserActive(targetNum, true)
		u.syncSearchActive(targetID)
		return nil, ErrNotBanned
	}
	if err != nil {
		return nil, err
	}

	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	if err := writer.SetUserActive(targetNum, true); err != nil {
		return nil, err
	}

	unbannedBy := callerID
	if unbannedBy == "" {
		unbannedBy = "system"
	}
	closed, err := u.bans.CloseBan(active.ID, unbannedBy, reason)
	if err != nil {
		return nil, err
	}
	u.syncSearchActive(targetID)
	return closed, nil
}

func (u *BanUseCaseImpl) GetActiveBan(lmsUserID string) (*models.UserBanCore, error) {
	ban, err := u.bans.GetActiveBan(lmsUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return ban, err
}

func (u *BanUseCaseImpl) ListBanHistory(lmsUserID string, limit int) ([]*models.UserBanCore, error) {
	return u.bans.ListBanHistory(lmsUserID, limit)
}

func (u *BanUseCaseImpl) ProcessExpiredBans() (int, error) {
	now := time.Now().UTC()
	expired, err := u.bans.ListExpiredActiveBans(now, 100)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, ban := range expired {
		if _, err := u.UnbanUser("system", ban.LmsUserID, "expired"); err != nil {
			if errors.Is(err, ErrNotBanned) {
				continue
			}
			log.Printf("moderation: auto-unban %s: %v", ban.LmsUserID, err)
			continue
		}
		count++
	}
	return count, nil
}
