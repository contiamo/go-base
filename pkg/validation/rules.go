package validation

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

// JobCronFormat is the default allowed CRON formatting options, allowing granularity down to the
// minute and the shorthand descriptions like @hourly
var JobCronFormat = cron.Minute |
	cron.Hour |
	cron.Dom |
	cron.Month |
	cron.Dow |
	cron.Descriptor

const (
	// MinNameLength is the minimal length of a name
	MinNameLength = 2
	// MaxNameLength is the maximal length of a name
	MaxNameLength = 255
	// MaxSQLIdentifierLength is the max length of a sql name
	// that we can support
	MaxSQLIdentifierLength = 64
	sqlIdentifierErrorMsg  = "SQL names must start with an alphabetic character and may only include alphanumeric characters and underscores '_'"
)

var identifierRE = regexp.MustCompile("^[a-zA-Z]+[a-zA-Z0-9_]*$")

// CronTab returns true if the string is a cron tab expression
func CronTab(value string) error {
	p := cron.NewParser(JobCronFormat)
	_, cerr := p.Parse(value)
	if cerr != nil {
		return errors.Wrap(cerr, "failed to parse crontab")
	}

	return nil
}

// Name returns an error if the name value is not valid
func Name(value string) error {
	return validation.Validate(
		value,
		validation.Required,
		validation.Length(MinNameLength, MaxNameLength),
	)
}

// SQLIdentifier returns an error if the string value can't be used as a SQL identifier
func SQLIdentifier(value string) error {
	return validation.Validate(
		value,
		validation.Required,
		validation.Length(MinNameLength, MaxSQLIdentifierLength),
		validation.Match(identifierRE).Error(sqlIdentifierErrorMsg),
	)
}

// URL returns an error if the URL value is not valid
func URL(value string) error {
	return validation.Validate(value, validation.Required, is.URL)
}

// UUID returns an error if the UUID value is not valid
func UUID(value string) error {
	return validation.Validate(value, validation.Required, is.UUID)
}

// SQLNameOrUUID returns an error if the given value can not be a name or UUID
func SQLNameOrUUID(value string) (err error) {
	err = UUID(value)
	if err == nil {
		return nil
	}
	return SQLIdentifier(value)
}

// NotEmpty checks if the value is not empty
func NotEmpty(value string) error {
	return validation.Validate(value, validation.NilOrNotEmpty)
}

// UUIDs returns an error if one of the UUID values is not valid
func UUIDs(values []string) error {
	err := validation.Validate(values, validation.Required, validation.Length(1, 0))
	if err != nil {
		return err
	}
	errors := validation.Errors{}
	for _, val := range values {
		errors[val] = UUID(val)
	}

	return errors.Filter()
}
