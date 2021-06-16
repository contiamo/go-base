package test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/v4/pkg/config"
	cdb "github.com/contiamo/go-base/v4/pkg/db"

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
func EqualCount(t *testing.T, db *sql.DB, expected int, table string, filter squirrel.Sqlizer, msgAndArgs ...interface{}) int {
	var count int
	err := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("COUNT(*)").
		From(table).
		Where(filter).
		RunWith(db).
		Scan(&count)
	require.NoError(t, err, msgAndArgs...)
	require.Equal(t, expected, count, msgAndArgs...)

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

// cleanupDB attempts to remove the test container created by EnsureDBReady
func cleanupDB(t *testing.T) {
	cmd := exec.Command("docker", "rm", "-v", "-f", "go-base-postgres")
	err := cmd.Run()
	if err != nil {
		t.Logf("error during db cleanupdb: %s", err)
	}
}

// EnsureDBReady is a helper utility that can be inserted into a test during development to make it simpler
// to run the test in isolation. Docker will be called to start a db that is read  to be used with GetDatabase.
// Example usage:
//
//   dberr, cleanup := dbtest.EnsureDBReady(ctx)
//   require.NoError(t, dberr)
//   defer cleanup()
//
//   _, db := dbtest.GetDatabase(t)
//   defer db.Close()
//
// While the code is somewhat robust to having existing dbs running, this is not intended to be left in the test code.
// The goal is that you can run individual tests via your IDE integrations or using the CLI, e.g.
//   go test -run ^TestRetentionHandler$`
func EnsureDBReady(ctx context.Context) (func(*testing.T), error) {
	check := exec.CommandContext(ctx, "docker", "container", "inspect", "go-base-postgres", "-f", "{{.ID}}").Run()
	if check == nil {
		// container is already running
		return func(*testing.T) {}, nil
	}

	if exitError, ok := check.(*exec.ExitError); ok && exitError.ExitCode() != 1 {
		return func(*testing.T) {}, fmt.Errorf("unexpected exit code when checking for running db: %w", check)
	}

	db := exec.CommandContext(ctx,
		"docker",
		"run",
		"--rm",
		"-d",
		"--name", "go-base-postgres",
		"-p", "0.0.0.0:5432:5432",
		"-e", "POSTGRES_PASSWORD=localdev",
		"-e", "POSTGRES_USER=contiamo_test",
		"-e", "POSTGRES_DB=postgres",
		"postgres:alpine",
		"-c", "fsync=off",
		"-c", "full_page_writes=off",
		"-c", "synchronous_commit=off",
	)

	err := db.Run()

	time.Sleep(3 * time.Second)
	return cleanupDB, err
}
