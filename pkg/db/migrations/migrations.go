package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/contiamo/go-base/v2/pkg/crypto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMigrater creates a migration command that will execut the given list of migrations
func NewMigrater(stmts []string, assets http.FileSystem) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		return migrate(ctx, db, stmts, assets)
	}
}

func migrate(ctx context.Context, db *sql.DB, list []string, assets http.FileSystem) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logrus.Errorf("failed to start migration transaction: %v", err)
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}

		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = errors.Wrap(err, rollbackErr.Error())
		}

		logrus.Error(err)
	}()

	for _, stmtName := range list {
		logger := logrus.WithField("stmt", stmtName)
		logger.Debug("migration started")

		stmt, err := getSQL(stmtName, migrations, assets)
		if err != nil {
			logger.Errorf("migration failed: %v", err)
			return fmt.Errorf("migration failed: %w", err)
		}

		hash, err := crypto.HashToString(stmt)
		if err != nil {
			logger.Errorf("init stmt failed: %v", err)
			return fmt.Errorf("init stmt failed: %w", err)
		}

		sqlVersion := fmt.Sprintf("%s_%s", stmtName, hash)
		row := db.QueryRowContext(ctx,
			`SELECT version FROM migrations WHERE version = $1;`,
			sqlVersion,
		)

		var version string
		err = row.Scan(&version)
		if err != nil && err != sql.ErrNoRows {
			logger.Errorf("version hash scan err: %s", err.Error())
			return err
		}

		if version == sqlVersion {
			logger.Info("migration already run")
			continue
		}

		_, err = tx.ExecContext(ctx, stmt)
		if err != nil {
			logger.Errorf("migration failed: %v", err)
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return err
}
