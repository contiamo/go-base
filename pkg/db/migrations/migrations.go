package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/contiamo/go-base/v3/pkg/crypto"
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
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logrus.Errorf("failed to start migration transaction: %v", err)
		return err
	}

	logger := logrus.WithField("method", "migrate")

	defer func() {
		if err == nil {
			err = tx.Commit()
			if err != nil {
				logger.WithError(err).Error("can not commit migrations transaction")
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

	for _, stmtName := range list {
		logger := logrus.
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
			continue
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
		logger.WithField("affected", affected).Info("migration finished")
	}

	return err
}
