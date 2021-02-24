package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/v3/pkg/queue/postgres"
	qpostgres "github.com/contiamo/go-base/v3/pkg/queue/postgres"
	"github.com/sirupsen/logrus"
)

// QueueDBConfig hold the db configuration values required to initialize
// the db tables for the queue. Specifically, it holds the definitions for
// any required foreign keys.
type QueueDBConfig struct {
	References []postgres.ForeignReference
}

// NewIniter creates a db init command that will execute the 000_init.sql
//
// The assets FileSystem must contain `migrations/000_init.sql`.
//
// If  queueConfig is nil, then the postgres queue will not be initialized.
func NewIniter(assets http.FileSystem, queueConfig *QueueDBConfig) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		return backoff.Retry(
			func() error {
				return initialize(ctx, db, assets, queueConfig)
			},
			backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
		)
	}
}

func initialize(ctx context.Context, db *sql.DB, assets http.FileSystem, queueConfig *QueueDBConfig) (err error) {
	logrus.Info("starting initialize attempt")
	logger := logrus.WithField("stmt", "init.sql")

	stmt, err := getSQL("000_init.sql", migrations, assets)
	if err != nil {
		return fmt.Errorf("can not read init statement: %w", err)
	}

	_, err = db.ExecContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("init execution failed: %w", err)
	}

	if queueConfig == nil {
		logger.Info("queue initialize skipped")
		return nil
	}

	// setup queue tables ('tasks' and 'schedules') and setup cascading delete for references
	logger.Info("queue initialize attempt")
	err = qpostgres.SetupTables(ctx, db, queueConfig.References)
	if err != nil {
		return fmt.Errorf("queue initialization failed: %w", err)
	}
	logger.Info("initialize completed")
	return nil
}
