package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// sqlKind is is used to distinguish where the SQL file opener will look for the SQL
type sqlKind string

const (
	migrations         sqlKind = "migrations"
	views              sqlKind = "views"
	idleTransactionErr         = "pq: unexpected transaction status idle"
)

func getSQL(name string, kind sqlKind, assets http.FileSystem) (string, error) {
	file, err := assets.Open(filepath.Join("/", string(kind), name))
	if err != nil {
		return "", fmt.Errorf("getSQL failed: %w", err)
	}

	s, err := ioutil.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("getSQL failed: %w", err)
	}

	return string(s), nil
}

func runStatement(ctx context.Context, db *sql.DB, stmt string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logrus.Errorf("failed to start migration transaction: %v", err)
		return err
	}

	logger := logrus.WithField("method", "runStatement")

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
			if err != nil && err.Error() == idleTransactionErr {
				logger.WithError(err).Warn("idle transaction at Commit")
				err = nil
			}
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

	return nil
}
