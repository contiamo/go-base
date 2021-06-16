package managers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	dbtest "github.com/contiamo/go-base/v4/pkg/db/test"
	"github.com/contiamo/go-base/v4/pkg/http/parameters"
	"github.com/stretchr/testify/require"
)

const testTableName = "base_manager_test_table"

func TestGetPageInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, db := dbtest.GetDatabase(t, createFixture)
	defer db.Close()

	cases := []struct {
		name     string
		table    string
		page     parameters.Page
		scope    squirrel.Sqlizer
		filter   squirrel.Sqlizer
		expected PageInfo
		expErr   string
	}{
		{
			name:   "Returns the valid page info when the scope and filter are set",
			table:  testTableName,
			page:   parameters.Page{Number: 1, Size: 5},
			scope:  squirrel.Eq{"organization_id": 1},
			filter: squirrel.Eq{"cats_involved": true},
			expected: PageInfo{
				ItemsPerPage:        5,
				Current:             1,
				UnfilteredItemCount: 8,
				ItemCount:           4,
			},
		},
		{
			name:  "Returns the valid page info when only the scope is set",
			table: testTableName,
			page:  parameters.Page{Number: 1, Size: 5},
			scope: squirrel.Eq{"organization_id": 1},
			expected: PageInfo{
				ItemsPerPage:        5,
				Current:             1,
				UnfilteredItemCount: 8,
				ItemCount:           8,
			},
		},
		{
			name:   "Returns the valid page info when only the filter is set",
			table:  testTableName,
			page:   parameters.Page{Number: 1, Size: 5},
			filter: squirrel.Eq{"cats_involved": true},
			expected: PageInfo{
				ItemsPerPage:        5,
				Current:             1,
				UnfilteredItemCount: 16,
				ItemCount:           8,
			},
		},
		{
			name:  "Returns the valid page info when neither filter nor scope is set",
			table: testTableName,
			page:  parameters.Page{Number: 1, Size: 5},
			expected: PageInfo{
				ItemsPerPage:        5,
				Current:             1,
				UnfilteredItemCount: 16,
				ItemCount:           16,
			},
		},
		{
			name:   "Returns error when the table name is wrong",
			table:  "wrong",
			page:   parameters.Page{Number: 1, Size: 5},
			expErr: "pq: relation \"wrong\" does not exist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewBaseManager(db, "test")
			info, err := m.GetPageInfo(ctx, tc.table, tc.page, tc.scope, tc.filter)
			if tc.expErr != "" {
				require.Error(t, err)
				require.Equal(t, tc.expErr, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, info)
		})
	}
}

func createFixture(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "CREATE TABLE "+testTableName+`(
		id integer PRIMARY KEY,
		organization_id integer NOT NULL,
    name text NOT NULL,
		description text NOT NULL,
    cats_involved boolean NOT NULL);`)

	if err != nil {
		return err
	}

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db).
		Insert(testTableName).
		Columns("id", "organization_id", "name", "description", "cats_involved")

	// 2 orgs with 8 values in each: 4 where cats are not involved and 4 where they are
	for i := 0; i < 2; i++ {
		for j := 0; j < 4; j++ {
			builder = builder.Values(
				10*i+j,
				i,
				fmt.Sprintf("name-%d", j),
				fmt.Sprintf("decr-%d", j),
				true,
			)
		}
		for j := 4; j < 8; j++ {
			builder = builder.Values(
				10*i+j,
				i,
				fmt.Sprintf("name-%d", j),
				fmt.Sprintf("decr-%d", j),
				false,
			)
		}
	}

	_, err = builder.ExecContext(ctx)

	return err
}
