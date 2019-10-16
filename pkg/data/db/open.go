package db

import (
	"context"
	"database/sql"

	"github.com/cenkalti/backoff"

	// import postgres driver
	_ "github.com/lib/pq"
)

// Open opens a postgres database and retries until ctx.Done()
func Open(ctx context.Context, connStr string) (db *sql.DB, err error) {
	err = backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			{
				return backoff.Permanent(ctx.Err())
			}
		default:
			{
				db, err = sql.Open("postgres", connStr)
				return err
			}
		}
	}, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}
	return db, nil
}
