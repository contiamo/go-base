/*
Package migrations provides standardized database migration tools.

Typical usage of the package requires a little bit of preparation
in your project. The goal is that you will be able to write SQL files
that are then bundled in your final compiled binary automatically. We
use vfsgendev for this.

The final file system may look like this

	.
	├── assets_dev.go
	├── assets_vfsdata.go
	├── migrations.go
	├── migrations
	│   ├── 000_init.sql
	│   ├── 001_helper_functions.sql
	│   ├── 002_move_bigint_to_timestamptz.sql
	│   ├── 003_hash_tokens.sql
	│   ├── 004_service_account_creator.sql
	│   ├── 005_retention_task_upsert.sql
	│   ├── 006_project_timestamps.sql
	│   ├── 007_project_stats.sql
	│   └── 008_hard_delete_projects.sql
	└── views
	   └── 001_views.sql

The migrations and views folders will contain the raw SQL files for setting up and
migrating your application. They are required to exist.

Additionally, you will need to setup the assets_dev.go and migrations.go files.

	// assets_dev.go

	// +build dev
	package db

	import (
		"net/http"

		"github.com/contiamo/go-base/v2/pkg/fileutils/union"
	)

	// Assets contains the static SQL file assets for setup and migrations
	var Assets = migrations.NewSQLAssets(migrations.SQLAssets{
		Migrations: http.Dir("migrations"),
		Views:      http.Dir("views"),
	})

and then

	// migrations.go

	package db

	import "github.com/contiamo/app/pkg/config"

	var dbConfig = migrations.MigrationConfig{
		MigrationStatements: []string{
			"001_helper_functions.sql",
			"002_move_bigint_to_timestamptz.sql",
			"003_hash_tokens.sql",
			"004_service_account_creator.sql",
			"005_retention_task_upsert.sql",
			"006_project_timestamps.sql",
			"007_project_stats.sql",
			"008_hard_delete_projects.sql",
		},
		ViewStatements: []string{},
		Assets: Assets,
	}

	var PrepareDatabase = migrations.NewPrepareDatabase(dbConfig, config.Version)
	var Init = migrations.NewIniter(Assets)
	var ConfigureViews = migrations.NewPostIniter(dbConfig.ViewStatements, Assets)
*/
package migrations
