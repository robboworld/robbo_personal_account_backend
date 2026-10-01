package db_client

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ErrLegacyPostgresDisabled is returned by every query on the legacy Postgres client when
// legacyPostgres.enabled is false (the default two-database mode).
var ErrLegacyPostgresDisabled = errors.New("legacy postgres is disabled (legacyPostgres.enabled=false)")

// disabledConnPool never reaches a database: every call fails fast.
type disabledConnPool struct{}

func (disabledConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, ErrLegacyPostgresDisabled
}

func (disabledConnPool) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, ErrLegacyPostgresDisabled
}

func (disabledConnPool) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, ErrLegacyPostgresDisabled
}

func (disabledConnPool) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return nil
}

func (disabledConnPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	return nil, ErrLegacyPostgresDisabled
}

// newDisabledDB returns a gorm handle on which every operation and transaction fails with
// ErrLegacyPostgresDisabled. Legacy-only gateways (units, groups, courses, users) used to get
// a nil *gorm.DB and panic with a nil pointer dereference on the first call.
func newDisabledDB() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: disabledConnPool{}}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	reject := func(tx *gorm.DB) { _ = tx.AddError(ErrLegacyPostgresDisabled) }
	cb := db.Callback()
	for _, reg := range []error{
		cb.Create().Before("gorm:create").Register("robbo:legacy_disabled", reject),
		cb.Query().Before("gorm:query").Register("robbo:legacy_disabled", reject),
		cb.Update().Before("gorm:update").Register("robbo:legacy_disabled", reject),
		cb.Delete().Before("gorm:delete").Register("robbo:legacy_disabled", reject),
		cb.Row().Before("gorm:row").Register("robbo:legacy_disabled", reject),
		cb.Raw().Before("gorm:raw").Register("robbo:legacy_disabled", reject),
	} {
		if reg != nil {
			return nil, reg
		}
	}
	return db, nil
}
