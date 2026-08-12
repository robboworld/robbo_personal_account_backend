package gateway

import (
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/streak"
	"go.uber.org/fx"
)

// StreakGatewayImpl stores streak in LMS auth_userprofile.meta JSON.
type StreakGatewayImpl struct{}

type StreakGatewayModule struct {
	fx.Out
	streak.Gateway
}

func SetupStreakGateway(_ db_client.PostgresClient) StreakGatewayModule {
	return StreakGatewayModule{Gateway: &StreakGatewayImpl{}}
}

func (g *StreakGatewayImpl) GetByUserID(lmsUserID string) (*models.UserLoginStreakDB, error) {
	id, err := lmsdb.ParseLMSUserID(lmsUserID)
	if err != nil || id <= 0 {
		return nil, nil
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	meta, err := reader.GetLoginStreak(id)
	if err != nil {
		return nil, err
	}
	return metaToRow(lmsUserID, meta), nil
}

func (g *StreakGatewayImpl) Upsert(row *models.UserLoginStreakDB) error {
	if row == nil {
		return nil
	}
	id, err := lmsdb.ParseLMSUserID(row.LmsUserID)
	if err != nil || id <= 0 {
		return nil
	}
	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return err
	}
	defer writer.Close()

	return writer.UpsertLoginStreak(id, rowToMeta(row))
}

func metaToRow(lmsUserID string, meta lmsdb.LoginStreakMeta) *models.UserLoginStreakDB {
	row := &models.UserLoginStreakDB{
		LmsUserID:     strings.TrimSpace(lmsUserID),
		CurrentStreak: meta.Current,
		LongestStreak: meta.Longest,
		Timezone:      meta.Timezone,
	}
	if t, ok := lmsdb.ParseLoginDate(meta.LastLoginDate); ok {
		row.LastLoginDate = &t
	}
	return row
}

func rowToMeta(row *models.UserLoginStreakDB) lmsdb.LoginStreakMeta {
	meta := lmsdb.LoginStreakMeta{
		Current:  row.CurrentStreak,
		Longest:  row.LongestStreak,
		Timezone: row.Timezone,
	}
	if row.LastLoginDate != nil {
		meta.LastLoginDate = lmsdb.FormatLoginDate(*row.LastLoginDate)
	}
	return meta
}
