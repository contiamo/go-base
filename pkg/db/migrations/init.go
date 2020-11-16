package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/v2/pkg/queue/postgres"
	qpostgres "github.com/contiamo/go-base/v2/pkg/queue/postgres"
	"github.com/sirupsen/logrus"
)

// NewIniter creates a db init command that will execut the 000_init.sql
func NewIniter(assets http.FileSystem) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		return backoff.Retry(
			func() error {
				return initialize(ctx, db, assets)
			},
			backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
		)
	}
}

func initialize(ctx context.Context, db *sql.DB, assets http.FileSystem) (err error) {
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

	// setup queue tables ('tasks' and 'schedules') and setup cascading delete for references
	logger.Info("queue initialize attempt")
	err = qpostgres.SetupTables(ctx, db, []postgres.ForeignReference{
		// To add a new reference you have to write a separate migration.
		// Once this table structure created for the first time, it will never be modified
		{
			ColumnName:       "message_id",
			ColumnType:       "UUID",
			ReferencedTable:  "messages",
			ReferencedColumn: "message_id",
		},
	})
	if err != nil {
		return fmt.Errorf("queue initialization failed: %w", err)
	}
	logger.Info("initialize completed")
	return nil
}
