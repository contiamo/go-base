package managers

import (
	"context"
	"database/sql"
	"io/ioutil"
	"testing"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	dbtest "github.com/contiamo/go-base/pkg/db/test"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func init() {
	logrus.SetOutput(ioutil.Discard)
}

func Test_Sqlizer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, db := dbtest.GetDatabase(t)
	defer db.Close()

	r := NewIDResolver("test", "id", "name")
	manager := NewBaseManager(db, "id_resolver_test")
	builder := manager.GetQueryBuilder()
	id := uuid.NewV4().String()
	where, err := r.Sqlizer(ctx, builder, id, squirrel.Eq{
		"some": "cool",
	})
	require.NoError(t, err)
	sql, _, err := where.ToSql()
	require.NoError(t, err)
	require.Equal(t, "(some = ? AND id = ?)", sql)
}

func Test_Resolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t, func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, `CREATE TABLE test(
		id UUID PRIMARY KEY,
    parent_id UUID,
		name text,
		UNIQUE(name, parent_id)
	);`)
		return err
	})
	defer db.Close()

	secIDs := []uuid.UUID{
		uuid.NewV4(),
		uuid.NewV4(),
	}
	ids := []uuid.UUID{
		uuid.NewV4(),
		uuid.NewV4(),
		uuid.NewV4(),
	}

	_, err := db.ExecContext(ctx, `INSERT INTO test (id, parent_id, name) VALUES ($1,$2, 'unique')`, ids[0], secIDs[0])
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO test (id, parent_id, name) VALUES ($1,$2, 'regular')`, ids[1], secIDs[0])
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO test (id, parent_id, name) VALUES ($1,$2, 'regular')`, ids[2], secIDs[1])
	require.NoError(t, err)

	r := NewIDResolver("test", "id", "name")

	cases := []struct {
		name,
		value,
		expectedID string
		where  squirrel.Sqlizer
		expErr bool
	}{
		{
			name:       "Resolves UUID into ID",
			value:      ids[0].String(),
			expectedID: ids[0].String(),
			where:      nil,
		},
		{
			name:       "Resolves a unique name into ID",
			value:      "unique",
			expectedID: ids[0].String(),
			where:      nil,
		},
		{
			name:       "Resolves a non-unique name into ID for different parent IDs",
			value:      "regular",
			expectedID: ids[1].String(),
			where: squirrel.Eq{
				"parent_id": secIDs[0],
			},
		},
		{
			name:       "Resolves a second non-unique name into ID",
			value:      "regular",
			expectedID: ids[2].String(),
			where: squirrel.Eq{
				"parent_id": secIDs[1],
			},
		},
		{
			name:       "Triggers error when resolve a non-existent name",
			value:      "wrong",
			expectedID: "",
			where: squirrel.Eq{
				"parent_id": secIDs[0],
			},
			expErr: true,
		},
		{
			name:       "Triggers error when there are more than one result",
			value:      "regular",
			expectedID: "",
			expErr:     true,
		},
		{
			name:   "Triggers validation error when the value is empty",
			expErr: true,
		},
	}

	for _, tc := range cases {
		manager := NewBaseManager(db, "id_resolver_test")
		builder := manager.GetQueryBuilder()
		t.Run(tc.name, func(t *testing.T) {
			id, err := r.Resolve(ctx, builder, tc.value, tc.where)
			if tc.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedID, id)
		})
	}
}
