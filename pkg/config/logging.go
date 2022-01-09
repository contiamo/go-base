package config

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Logging configures the log level and formatter, if the formatter is nil,
// the default TextFormatter is used.
func Logging(level string, formatter logrus.Formatter) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("can not parse log-level: %w", err)
	}

	if formatter == nil {
		formatter = &logrus.TextFormatter{
			// DisableColors: true,
			FullTimestamp:          true,
			DisableLevelTruncation: true,
			PadLevelText:           true,
		}
	}

	logrus.SetLevel(lvl)
	logrus.SetFormatter(formatter)

	return nil
}
