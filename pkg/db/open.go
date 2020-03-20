package db

import (
	"context"
	"database/sql"

	"github.com/cenkalti/backoff"
	"github.com/contiamo/go-base/pkg/config"
)

// Open opens a connection to a database and retries until ctx.Done()
// The users must import all the necessary drivers before calling this function.
func Open(ctx context.Context, cfg config.Database) (db *sql.DB, err error) {
	connStr, err := cfg.GetConnectionString()

	err = backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			{
				return backoff.Permanent(ctx.Err())
			}
		default:
			{
				db, err = sql.Open(cfg.DriverName, connStr)
				return err
			}
		}
	}, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.PoolSize)
	return db, nil
}
