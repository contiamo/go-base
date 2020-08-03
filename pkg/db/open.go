package db

import (
	"context"
	"database/sql"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/pkg/config"
	"github.com/sirupsen/logrus"
)

// Open opens a connection to a database and retries until ctx.Done()
// The users must import all the necessary drivers before calling this function.
func Open(ctx context.Context, cfg config.Database) (db *sql.DB, err error) {
	connStr, err := cfg.GetConnectionString()
	if err != nil {
		return nil, err
	}

	err = backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			{
				return backoff.Permanent(ctx.Err())
			}
		default:
			{
				db, err = sql.Open(cfg.DriverName, connStr)
				if err != nil {
					logrus.Errorf("failed to open db connection: %s", err)
					return err
				}

				err = db.Ping()
				if err != nil {
					logrus.Errorf("failed to ping target db: %s", err)
					return err
				}

				return nil
			}
		}
	}, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.PoolSize)
	return db, nil
}
