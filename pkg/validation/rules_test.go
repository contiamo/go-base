package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCronTab(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, CronTab("* * * * *"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := CronTab("invalid")
		require.Error(t, err)
		require.Equal(t, "failed to parse crontab: Expected exactly 5 fields, found 1: invalid", err.Error())
	})
}

func TestName(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, Name("name"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := Name("a")
		require.Error(t, err)
		require.Equal(t, "the length must be between 2 and 255", err.Error())
	})
}

func TestSQLIdentifier(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, SQLIdentifier("name"))
	})
	t.Run("Returns error for names over 63 characters", func(t *testing.T) {
		tooLong := "a123456789_123456789_123456789_123456789_123456789_123456789_123"
		require.Error(t, SQLIdentifier(tooLong), len(tooLong))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := SQLIdentifier("n@me")
		require.Error(t, err)
		require.Equal(t, "SQL names must start with an alphabetic character and may only include alphanumeric characters and underscores '_'", err.Error())
	})
}

func TestURL(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, URL("http://example.com"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := URL("url")
		require.Error(t, err)
		require.Equal(t, "must be a valid URL", err.Error())
	})
}

func TestUUID(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, UUID("123e4567-e89b-12d3-a456-426655440000"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := UUID("not-uuid")
		require.Error(t, err)
		require.Equal(t, "must be a valid UUID", err.Error())
	})
}

func TestSQLNameOrUUID(t *testing.T) {
	t.Run("Returns no error if the value is valid UUID", func(t *testing.T) {
		require.NoError(t, SQLNameOrUUID("123e4567-e89b-12d3-a456-426655440000"))
	})
	t.Run("Returns no error if the value is valid name", func(t *testing.T) {
		require.NoError(t, SQLNameOrUUID("name"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := SQLNameOrUUID("neither-uuid-nor-name")
		require.Error(t, err)
		require.Equal(t, "must be a valid UUID or a valid SQL identifier: SQL names must start with an alphabetic character and may only include alphanumeric characters and underscores '_'", err.Error())
	})
}

func TestNotEmpty(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, NotEmpty("something"))
	})
	t.Run("Returns error if the value is invalid", func(t *testing.T) {
		err := NotEmpty("")
		require.Error(t, err)
		require.Equal(t, "cannot be blank", err.Error())
	})
}

func TestUUIDs(t *testing.T) {
	t.Run("Returns no error if the value is valid", func(t *testing.T) {
		require.NoError(t, UUIDs([]string{
			"123e4567-e89b-12d3-a456-426655440000",
			"123e4567-e89b-12d3-a456-426655440001",
		}))
	})
	t.Run("Returns error if the list is empty", func(t *testing.T) {
		err := UUIDs([]string{})
		require.Error(t, err)
		require.Equal(t, "cannot be blank", err.Error())
	})
	t.Run("Returns error if some list items are invalid", func(t *testing.T) {
		err := UUIDs([]string{
			"123e4567-e89b-12d3-a456-426655440000",
			"invalid",
			"another",
		})
		require.Error(t, err)
		require.Equal(t, "another: must be a valid UUID; invalid: must be a valid UUID.", err.Error())
	})
}
