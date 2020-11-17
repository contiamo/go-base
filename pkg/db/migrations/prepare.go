package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/contiamo/go-base/v2/pkg/crypto"
	"github.com/sirupsen/logrus"
)

// MigrationConfig contains the ordered migration and view sql statements
// that will be run during startup as well as the assets filesystem object.
// Use NewSQLAssets to generate this filesystem object.
type MigrationConfig struct {
	MigrationStatements []string
	ViewStatements []string
	Assets http.FileSystem
}

// NewPrepareDatabase is the standard entrypoint for setting up and migrating the application db.
// This command will ensure that migration tracking is configured, check for any required migration
// commands that need to be run, and then update the migration tracking table.
//
// You can inspect the migrations table of your app using
//
// 	select * from migrations;
// 	select * from migrations WHERE applied_at > '<timestamp>';
// 	select * from mgirations WHERE version = '<app version>';
//
// The migration tracking will track _both_ the individual migrations (using a hash of the sql) _and_
// the migrations in a specific version of the app. If the app version is found in the migration history
// then it assumes that all of the required migrations have been run and exists early.
//
// To force a migration to rerun, you will need to delete the record from the tracking table
//
// 	delete from migrations where version = '<migration version>';
func NewPrepareDatabase(config MigrationConfig, queueConfig *QueueDBConfig, appVersion string) func(context.Context, *sql.DB) error {
	initDB := NewIniter(config.Assets, queueConfig)
	migrateDB := NewMigrater(config.MigrationStatements, config.Assets)
	setupViews := NewPostIniter(config.ViewStatements, config.Assets)

	return func(ctx context.Context, database *sql.DB) (err error) {
		logger := logrus.WithField("version", appVersion)

		logger.Debug("preparing migration tracking")
		// initialize the migration list if does not exist.
		_, err = database.ExecContext(ctx,
			`CREATE TABLE IF NOT EXISTS migrations(
				version TEXT PRIMARY KEY,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
		)
		if err != nil {
			return err
		}

		// we always query the last migration, makes sense to have the index
		_, err = database.ExecContext(ctx,
			`CREATE INDEX IF NOT EXISTS migrations_applied_at_idx ON migrations (applied_at DESC);`,
		)
		if err != nil {
			return err
		}

		logger.Debug("start migration transaction")
		// open a transaction to aquire the lock, so several migrations can't run at the same time
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		defer func() {
			if err != nil {
				_ = tx.Rollback()
				return
			}
			err = tx.Commit()
			if err != nil {
				logger.Infof("migration finished with: %s", err)
			}
		}()

		// SHARE ROW EXCLUSIVE allows other sessions to read the table but nothing else
		logger.Info("waiting for migration lock")
		_, err = tx.ExecContext(ctx, `LOCK TABLE migrations IN SHARE ROW EXCLUSIVE MODE;`)
		if err != nil {
			return fmt.Errorf("migrations table locked: %s", err)
		}

		row := tx.QueryRowContext(ctx,
			`SELECT version FROM migrations WHERE version = $1;`, appVersion,
		)

		var version string
		err = row.Scan(&version)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("version scan err: %s", err)
		}

		// run the migrations on stringly if they have not been run before
		if version == appVersion {
			logger.Info("Database version match, no migration required...")
			return nil
		}

		logger.Info("Running SQL initialization statements...")
		// don't do this check inside the Init because we call this inside all of our tests.
		// the alternative is to setup the migration table in every test case, this produces the
		// smallest change.
		run, err := shouldRunInit(ctx, config.Assets, database)
		if err != nil {
			return err
		}
		if run {
			err = initDB(ctx, database)
			if err != nil {
				return fmt.Errorf("init err: %s", err)
			}

			err = saveVersionHash(ctx, "000_init.sql", config.Assets, tx)
			if err != nil {
				return err
			}
		}
		logger.Info("Finished running SQL initialization statements...")

		logger.Info("Running SQL migration statements...")
		err = migrateDB(ctx, database)
		if err != nil {
			return fmt.Errorf("migrate err: %s", err)
		}
		logger.Info("Finished running SQL migration statements...")

		logger.Info("Running View initialization statements...")
		err = setupViews(ctx, database)
		if err != nil {
			return fmt.Errorf("configure views err: %s", err)
		}
		logger.Info("Finished running View initialization statements...")

		for _, name := range config.MigrationStatements {
			err = saveVersionHash(ctx, name, config.Assets, tx)
			if err != nil {
				return err
			}
		}

		// store the migration into the log
		_, err = tx.ExecContext(ctx, `
			INSERT INT stringO migrations (version) VALUES ($1);`,
			appVersion,
		)
		if err != nil {
			return err
		}

		return nil
	}
}

func shouldRunInit(ctx context.Context, assets http.FileSystem, db *sql.DB) (bool, error) {
	stmt, err := getSQL("000_init.sql", migrations, assets)
	if err != nil {
		return false, fmt.Errorf("can not read init statement: %w", err)
	}

	hash, err := crypto.HashToString(stmt)
	if err != nil {
		return false, fmt.Errorf("init statement hash failed: %w", err)
	}

	sqlVersion := fmt.Sprintf("%s_%s", "000_init.sql", hash)
	row := db.QueryRowContext(ctx,
		`SELECT version FROM migrations WHERE version = $1;`,
		sqlVersion,
	)

	var version string
	err = row.Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("version hash scan err: %w", err)
	}

	if version == sqlVersion {
		logrus.Info("init already run")
		return false, nil
	}
	return true, nil
}

func saveVersionHash(ctx context.Context, name string, assets http.FileSystem, tx *sql.Tx) (err error) {
	stmt, err := getSQL(name, migrations, assets)
	if err != nil {
		return fmt.Errorf("can not load sql statement: %w", err)
	}

	hash, err := crypto.HashToString(stmt)
	if err != nil {
		return fmt.Errorf("sql statement hash failed: %w", err)
	}

	sqlVersion := fmt.Sprintf("%s_%s", name, hash)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING;`,
		sqlVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to save sql version: %w", err)
	}

	return nil
}
