package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/v4/pkg/config"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/sirupsen/logrus"
)

// Open opens a connection to a database and retries until ctx.Done()
// The users must import all the necessary drivers before calling this function.
func Open(ctx context.Context, cfg config.Database) (db *sql.DB, err error) {
	tracer := tracing.NewTracer("db", "Connection")
	span, ctx := tracer.StartSpan(ctx, "Open")
	defer func() {
		tracer.FinishSpan(span, err)
	}()

	span.SetTag("host", cfg.Host)
	span.SetTag("port", cfg.Port)
	span.SetTag("name", cfg.Name)
	span.SetTag("username", cfg.Username)

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
					errMsg := fmt.Sprintf("failed to open db connection: %s", err)
					span.LogKV("error", errMsg)
					logrus.Errorf(errMsg)
					return err
				}

				err = db.Ping()
				if err != nil {
					errMsg := fmt.Sprintf("failed to ping target db: %s", err)
					span.LogKV("error", errMsg)
					logrus.Errorf(errMsg)
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
