package migrations

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	dbtest "github.com/contiamo/go-base/v4/pkg/db/test"
	"github.com/contiamo/go-base/v4/pkg/queue/postgres"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestGetJitter(t *testing.T) {
	value := float64(GetJitter(time.Second))
	require.LessOrEqual(t, value, float64(time.Second))
	require.GreaterOrEqual(t, value, 0.05*float64(time.Second))
}
func TestInitAndMigrate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	assets := NewSQLAssets(SQLAssets{
		Migrations: http.Dir("./testdata/migrations"),
		Views:      http.Dir("./testdata/views"),
	})
	dbConfig := MigrationConfig{
		MigrationStatements: []string{
			"001_add_description.sql",
		},
		ViewStatements: []string{"001_ids_views.sql"},
		Assets:         assets,
	}

	cases := []struct {
		name  string
		queue *QueueDBConfig
	}{
		{"without queue", nil},
		{
			"with queue",
			&QueueDBConfig{
				References: []postgres.ForeignReference{
					{
						ColumnName:       "resource_id",
						ColumnType:       "UUID",
						ReferencedTable:  "resources",
						ReferencedColumn: "resource_id",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prepare := NewPrepareDatabase(dbConfig, tc.queue, "TestInitAndMigrate-"+tc.name)

			_, db := dbtest.GetDatabase(t, prepare)
			defer db.Close()

			builder := squirrel.StatementBuilder.
				PlaceholderFormat(squirrel.Dollar).
				RunWith(db)

			id := uuid.NewV4()
			query := builder.Insert("resources").Columns(
				"resource_id",
				"parent_id",
				"kind",
				"name",
				"description",
			).Values(
				id,
				id,
				"wood",
				"oak",
				"this is a tree",
			)

			_, err := query.ExecContext(ctx)
			require.NoError(t, err, "can not write to the new db:\n\n%s", squirrel.DebugSqlizer(query))

			var viewID string
			selecter := builder.Select("id").From("resource_ids").Limit(1)

			err = selecter.ScanContext(ctx, &viewID)
			require.NoError(t, err, "can not select from the view:\n\n%s", squirrel.DebugSqlizer(selecter))

			require.Equal(t, id.String(), viewID)

			err = prepare(ctx, db)
			require.NoError(t, err, "prepare should be idempotent")
		})
	}
}
