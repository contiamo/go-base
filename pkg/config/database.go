package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
)

var defaultPorts = map[string]uint32{
	"postgres": 5432,
}

const (
	// these environment variables can be used to control the SSL
	// validation behavior and behave the same as the documented
	// postegres environment variables
	PGSSLModeEnvKey         = "PGSSLMODE"
	PGSSLCertPathEnvKey     = "PGSSLCERT"
	PGSSLKeyPathEnvKey      = "PGSSLKEY"
	PGSSLRootCertPathEnvKey = "PGSSLROOTCERT"
)

const (
	pgSSLDisabled   = "disable"
	pgSSLRequire    = "require"
	pgSSLVerifyCA   = "verify-ca"
	pgSSLVerifyFull = "verify-full"
)

// Database contains all the configuration parameters for a database
// SSL mode and options can be configured via the standard Postgres env variables
// documented here https://www.postgresql.org/docs/current/libpq-envars.html
//
// Specifically, it supports:  PGSSLMODE, PGSSLCERT, PGSSLKEY, PGSSLROOTCERT.
type Database struct {
	// Host of the database server
	Host string `json:"host"`
	// Port of the database server, if empty, we will attempt to parse the port from the host value,
	// if Host does not have a port value, then we will use the default value from the db driver
	Port uint32 `json:"port"`
	// Name is the name of the database on the host
	Name string `json:"name"`
	// Username to access the database
	Username string `json:"username"`
	// PasswordPath is a path to the file where the password is stored
	PasswordPath string `json:"passwordPath"`
	// PoolSize is the max number of concurrent connections to the database,
	// <=0 is unlimited
	PoolSize int `json:"poolSize"`
	// DriverName is the database driver name e.g. postgres
	DriverName string `json:"driverName"`
}

// GetPassword gets the database password from PasswordPath
func (cfg *Database) GetPassword() (string, error) {
	if cfg.PasswordPath == "" {
		return "", nil
	}
	passwordBytes, err := ioutil.ReadFile(cfg.PasswordPath)
	if err != nil {
		return "", errors.Wrapf(err, "can not read the database password file `%s`", cfg.PasswordPath)
	}

	return strings.TrimSpace(string(passwordBytes)), nil
}

// GetHost returns the host name of the underlying db
func (cfg *Database) GetHost() string {
	if cfg.Host != "" {
		return cfg.Host
	}

	return "localhost"
}

// GetPort returns the port of the underlying db
func (cfg *Database) GetPort() uint32 {
	if cfg.Port != 0 {
		return cfg.Port
	}

	return defaultPorts[cfg.DriverName]
}

// GetConnectionString returns the formed connection string
func (cfg *Database) GetConnectionString() (connStr string, err error) {
	sslMode, found := os.LookupEnv(PGSSLModeEnvKey)
	if !found {
		sslMode = pgSSLDisabled
	}

	switch sslMode {
	case pgSSLDisabled, pgSSLRequire, pgSSLVerifyCA, pgSSLVerifyFull:
		// nothing to do
	default:
		return "", fmt.Errorf("unknown or unsupported ssl mode: %q", sslMode)
	}

	connStr = fmt.Sprintf("sslmode=%s ", sslMode)
	if value, found := os.LookupEnv(PGSSLCertPathEnvKey); found {
		connStr += fmt.Sprintf("sslcert=%s ", value)
	}

	if value, found := os.LookupEnv(PGSSLKeyPathEnvKey); found {
		connStr += fmt.Sprintf("sslkey=%s ", value)
	}

	if value, found := os.LookupEnv(PGSSLRootCertPathEnvKey); found {
		connStr += fmt.Sprintf("sslrootcert=%s ", value)
	}

	if cfg.Host != "" {
		connStr += fmt.Sprintf("host=%s ", cfg.Host)
	}

	if cfg.Port != 0 {
		connStr += fmt.Sprintf("port=%d ", cfg.Port)
	}

	if cfg.Name != "" {
		connStr += fmt.Sprintf("dbname=%s ", cfg.Name)
	}
	if cfg.Username != "" {
		connStr += fmt.Sprintf("user=%s ", cfg.Username)
	}
	if cfg.PasswordPath != "" {
		pw, err := cfg.GetPassword()
		if err != nil {
			return "", err
		}
		if pw != "" {
			connStr += fmt.Sprintf("password=%s ", pw)
		}
	}

	return strings.TrimSpace(connStr), nil
}
