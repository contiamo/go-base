package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/pkg/config"
	cdb "github.com/contiamo/go-base/pkg/db"
	"github.com/sirupsen/logrus"

	// since this test helper is going to be used in tests the CLI would not initialize
	// the drivers for us, so we need to put it here again
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	defaultDBName = "postgres" // in postgres the default DB is `postgres`
	adminUser     = "contiamo_test"
	adminPassword = "localdev"
)

// DBInitializer is the function that initializes the data base for testing
type DBInitializer func(context.Context, *sql.DB) error

// EqualCount asserts that the count of rows matches the expected value given the table and WHERE query and args.EqualCount
// Note that this is a simple  COUNT of rows in a single table. More complex queries should be constructed by hand.
func EqualCount(t *testing.T, db *sql.DB, expected int, table string, filter squirrel.Sqlizer) int {
	var count int
	err := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("COUNT(*)").
		From(table).
		Where(filter).
		RunWith(db).
		Scan(&count)
	require.NoError(t, err)
	require.Equal(t, expected, count)

	return count
}

// GetDatabase gets a test database
func GetDatabase(t *testing.T, inits ...DBInitializer) (name string, testDB *sql.DB) {
	var err error
	defer func() {
		if testDB == nil {
			t.Fatalf("failed to open connection to the test database: %s", err.Error())
		}
	}()

	name = cdb.GenerateSQLName()
	testDB, err = connectDB(name)
	if err != nil {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if inits == nil {
		return name, testDB
	}
	for _, init := range inits {
		// for backwards compatibility
		if init == nil {
			continue
		}
		err = init(ctx, testDB)
		if err != nil {
			t.Fatalf("Can't initialize the database `%s`: %v", name, err)
			return "", nil
		}
	}
	t.Cleanup(func() {
		if testDB != nil {
			if err := testDB.Close(); err != nil {
				logrus.Warnf("failed to close test db: %v", err)
			}
		}
	})
	return name, testDB
}

func connectDB(name string) (db *sql.DB, err error) {
	cfg := config.Database{
		Host:       "localhost",
		Name:       defaultDBName,
		Username:   adminUser,
		DriverName: "postgres",
	}
	adminConnStr, err := cfg.GetConnectionString()
	if err != nil {
		return nil, err
	}

	adminDB, err := sql.Open(cfg.DriverName, adminConnStr+" password="+adminPassword)
	if err != nil {
		return nil, err
	}
	defer adminDB.Close()
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	if err != nil {
		return nil, err
	}

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		return nil, err
	}

	cfg.Name = name
	connStr, err := cfg.GetConnectionString()
	if err != nil {
		return nil, err
	}

	return sql.Open(cfg.DriverName, connStr+" password="+adminPassword)
}
