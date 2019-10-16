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

	// since this test helper is going to be used in tests the CLI would not initialize
	// the drivers for us, so we need to put it here again
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	defaultDBName = "postgres" // in postgres the default DB is `postgres`
	adminUser     = "contiamo_test"
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
func GetDatabase(t *testing.T, init DBInitializer) (name string, testDB *sql.DB) {
	defer func() {
		if testDB == nil {
			t.Fatal("failed to open connection to the test database")
		}
	}()

	name = cdb.GenerateSQLName()
	testDB, err := connectDB(name)
	if err != nil {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = init(ctx, testDB)
	if err != nil {
		t.Fatalf("Can't initialize the database `%s`: %v", name, err)
		return "", nil
	}
	return name, testDB
}

func connectDB(name string) (db *sql.DB, err error) {
	cfg := config.Database{
		Host:         "0.0.0.0",
		Name:         defaultDBName,
		Username:     adminUser,
		PasswordPath: "./password",
		DriverName:   "postgres",
	}
	adminConnStr, err := cfg.GetConnectionString()
	if err != nil {
		return nil, err
	}

	adminDB, err := sql.Open(cfg.DriverName, adminConnStr)
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

	return sql.Open(cfg.DriverName, connStr)
}
