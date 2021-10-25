package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseGetPassword(t *testing.T) {
	t.Run("Returns the password when it's found", func(t *testing.T) {
		cfg := Database{PasswordPath: "./testdata/password"}
		password, err := cfg.GetPassword()
		require.NoError(t, err)
		require.Equal(t, "password", password)
	})
	t.Run("Returns no password when it's not set", func(t *testing.T) {
		cfg := Database{}
		password, err := cfg.GetPassword()
		require.NoError(t, err)
		require.Empty(t, password)
	})

	t.Run("Returns the error when the password is not found", func(t *testing.T) {
		cfg := Database{PasswordPath: "./testdata/invalid"}
		password, err := cfg.GetPassword()
		require.Error(t, err)
		require.Empty(t, password)
		require.Equal(t, "can not read the database password file `./testdata/invalid`: open ./testdata/invalid: no such file or directory", err.Error())
	})
}

func TestDatabaseGetHost(t *testing.T) {
	t.Run("Returns the host name when specified", func(t *testing.T) {
		host := "some.host"
		cfg := Database{Host: host}
		require.Equal(t, host, cfg.GetHost())
	})
	t.Run("Returns `localhost` when name is not specified", func(t *testing.T) {
		cfg := Database{}
		require.Equal(t, "localhost", cfg.GetHost())
	})
}

func TestDatabaseGetPort(t *testing.T) {
	t.Run("Returns specified port when it's set", func(t *testing.T) {
		var port uint32 = 80
		cfg := Database{Port: port}
		require.Equal(t, port, cfg.GetPort())
	})
	t.Run("Returns default port when it's not set", func(t *testing.T) {
		cfg := Database{DriverName: "postgres"}
		require.Equal(t, uint32(5432), cfg.GetPort())
	})
	t.Run("Returns 0 when the port is not set and the default value is not found", func(t *testing.T) {
		cfg := Database{}
		require.Equal(t, uint32(0), cfg.GetPort())
	})
}

func TestDatabaseGetConnectionString(t *testing.T) {
	cases := []struct {
		name     string
		cfg      Database
		env      map[string]string
		expected string
		err      string
	}{
		{
			name: "Returns the full connection string when all is set",
			cfg: Database{
				Host:         "example.com",
				Port:         80,
				Name:         "database",
				Username:     "root",
				PasswordPath: "./testdata/password",
			},
			expected: "sslmode=disable host=example.com port=80 dbname=database user=root password=password",
		},
		{
			name: "Can set sslmode to require via env variables",
			cfg: Database{
				Host:         "example.com",
				Port:         80,
				Name:         "database",
				Username:     "root",
				PasswordPath: "./testdata/password",
			},
			env: map[string]string{
				"PGSSLMODE":     "require",
				"PGSSLCERT":     "/cert/path",
				"PGSSLKEY":      "/key/path",
				"PGSSLROOTCERT": "/root/cert/path",
			},
			expected: "sslmode=require sslcert=/cert/path sslkey=/key/path sslrootcert=/root/cert/path host=example.com port=80 dbname=database user=root password=password",
		},
		{
			name: "Returns a connection string when Host is not set",
			cfg: Database{
				Port:         80,
				Name:         "database",
				Username:     "root",
				PasswordPath: "./testdata/password",
			},
			expected: "sslmode=disable port=80 dbname=database user=root password=password",
		},
		{
			name: "Returns a connection string when Port is not set",
			cfg: Database{
				Host:         "example.com",
				Name:         "database",
				Username:     "root",
				PasswordPath: "./testdata/password",
			},
			expected: "sslmode=disable host=example.com dbname=database user=root password=password",
		},
		{
			name: "Returns a connection string when Name is not set",
			cfg: Database{
				Host:         "example.com",
				Port:         80,
				Username:     "root",
				PasswordPath: "./testdata/password",
			},
			expected: "sslmode=disable host=example.com port=80 user=root password=password",
		},
		{
			name: "Returns a connection string when Username is not set",
			cfg: Database{
				Host:         "example.com",
				Name:         "database",
				Port:         80,
				PasswordPath: "./testdata/password",
			},
			expected: "sslmode=disable host=example.com port=80 dbname=database password=password",
		},
		{
			name: "Returns a connection string when PasswordPath is not set",
			cfg: Database{
				Host:     "example.com",
				Name:     "database",
				Port:     80,
				Username: "root",
			},
			expected: "sslmode=disable host=example.com port=80 dbname=database user=root",
		},
		{
			name: "Returns an error when PasswordPath is wrong",
			cfg: Database{
				Host:         "example.com",
				Name:         "database",
				Port:         80,
				Username:     "root",
				PasswordPath: "./testdata/invalid",
			},
			err: "can not read the database password file `./testdata/invalid`: open ./testdata/invalid: no such file or directory",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reset := envSetup(tc.env)
			defer reset()

			str, err := tc.cfg.GetConnectionString()
			if tc.err != "" {
				require.Error(t, err)
				require.Equal(t, tc.err, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, str)
		})
	}
}

func envSetup(envs map[string]string) (resetter func()) {
	if len(envs) == 0 {
		return func() {}
	}

	originalEnvs := map[string]string{}

	for name, value := range envs {
		if originalValue, ok := os.LookupEnv(name); ok {
			originalEnvs[name] = originalValue
		}
		_ = os.Setenv(name, value)
	}

	return func() {
		for name := range envs {
			origValue, has := originalEnvs[name]
			if has {
				_ = os.Setenv(name, origValue)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}
}
