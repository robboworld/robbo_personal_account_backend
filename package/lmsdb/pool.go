package lmsdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	poolsMu sync.Mutex
	pools   = map[string]*sql.DB{}
)

// sharedDB returns one connection pool per DSN for the whole process. Readers and writers
// used to open and ping a new pool on every call, including the per-request activity check.
// A failed connection is not cached, so the next call retries.
func sharedDB(dsn, label string) (*sql.DB, error) {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	if db := pools[dsn]; db != nil {
		return db, nil
	}
	db, err := sql.Open("mysql", ensureParseTimeDSN(dsn))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("%s ping: %w", label, err)
	}
	pools[dsn] = db
	return db, nil
}

// CloseAll closes the shared pools on shutdown.
func CloseAll(context.Context) error {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	var errs []error
	for dsn, db := range pools {
		errs = append(errs, db.Close())
		delete(pools, dsn)
	}
	return errors.Join(errs...)
}
