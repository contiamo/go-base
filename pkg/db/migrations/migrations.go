package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/contiamo/go-base/v4/pkg/crypto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMigrater creates a migration command that will execute the given list of migrations
func NewMigrater(stmts []string, assets http.FileSystem) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		return migrate(ctx, db, stmts, assets)
	}
}

func migrate(ctx context.Context, db *sql.DB, list []string, assets http.FileSystem) (err error) {
	for _, stmtName := range list {
		err = executeMigration(ctx, db, stmtName, assets)
		if err != nil {
			return err
		}
	}
	return nil
}

func executeMigration(ctx context.Context, db *sql.DB, stmtName string, assets http.FileSystem) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logrus.Errorf("failed to start migration transaction: %v", err)
		return err
	}

	logger := logrus.WithField("method", "migrate")

	defer func() {
		if err == nil {
			err = tx.Commit()
			// dirty ugly no-good hack!
			// we have not seen this in the wild yet, _but_ during our unit tests
			// we sometimes get this error, which indicates that the transaction  was
			// started (ie `BEGIN;`) but nothing has happened yet. For example:
			//    https://www.postgresql.org/message-id/20080617133250.GA68434@commandprompt.com
			//    https://github.com/lib/pq/issues/225
			// Initial experiments have only produced this during unit tests, but actual
			// application environments run without any transaction issues.
			if err != nil && err.Error() == "pq: unexpected transaction status idle" {
				logger.WithError(err).Warn("idle transaction at Commit")
				err = nil
			}
			if err != nil {
				logger.WithError(err).Error("can not commit migration transaction")
			}
			return
		}

		logger.WithError(err).Error("migration transaction requires rollback")

		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = errors.Wrap(err, rollbackErr.Error())
			logger.WithError(err).Error("migration rollback failed")
		}
	}()

	logger = logrus.
		WithField("method", "migrate").
		WithField("stmt", stmtName)
	logger.Debug("migration started")

	stmt, err := getSQL(stmtName, migrations, assets)
	if err != nil {
		logger.WithError(err).Error("migration failed")
		return fmt.Errorf("migration failed: %w", err)
	}

	hash, err := crypto.HashToString(stmt)
	if err != nil {
		logger.WithError(err).Error("can not hash sql statement")
		return errors.Wrap(err, "can not hash sql statement")
	}

	sqlVersion := fmt.Sprintf("%s_%s", stmtName, hash)
	row := db.QueryRowContext(ctx,
		`SELECT version FROM migrations WHERE version = $1;`,
		sqlVersion,
	)

	var version string
	err = row.Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		logger.WithError(err).Errorf("version hash scan err")
		return errors.Wrap(err, "version hash scan err")
	}

	if version == sqlVersion {
		logger.Info("migration already run")
		return nil
	}

	res, err := tx.ExecContext(ctx, stmt)
	if err != nil {
		logger.WithError(err).Error("migration failed")
		return errors.Wrap(err, "migration failed")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("can't count affected rows")
		return errors.Wrap(err, "can't count affected rows")
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING;`,
		sqlVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to save sql version: %w", err)
	}

	logger.WithField("affected", affected).Info("migration finished")

	return err
}
