package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewPostIniter creates a db init command that will execute the view sql, and other post init sql.
func NewPostIniter(stmts []string, assets http.FileSystem) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		return backoff.Retry(
			func() error {
				return configureViews(ctx, db, stmts, assets)
			},
			backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
		)
	}
}

func configureViews(ctx context.Context, db *sql.DB, stmts []string, assets http.FileSystem) error {
	for i, stmtName := range stmts {
		logger := logrus.WithField("stmt", stmtName)
		logger.Debug("migration started")

		stmt, err := getSQL(stmtName, views, assets)
		if err != nil {
			logger.Errorf("migration failed: %v", err)
			return fmt.Errorf("migration failed: %w", err)
		}
		err = runStatement(ctx, db, stmt)
		if err != nil {
			return errors.Wrapf(err, "configure view failed at index %d", i)
		}
	}

	return nil
}
