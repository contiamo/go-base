package testing

import (
	"bytes"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// SetupLoggingBuffer creates an empty buffer and sets it
// as a Logrus output globally. Returns the created buffer
// so you can check what was logged.
func SetupLoggingBuffer() (buf *bytes.Buffer, restore func()) {
	buf = bytes.NewBuffer(nil)
	std := logrus.StandardLogger()
	prevFormatter, prevOut, prevLevel := std.Formatter, std.Out, std.Level
	restore = func() {
		logrus.SetFormatter(prevFormatter)
		logrus.SetOutput(prevOut)
		logrus.SetLevel(prevLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	logrus.SetOutput(buf)
	logrus.SetLevel(logrus.DebugLevel)

	return buf, restore
}

// DiscardLogging sets the global Logrus output to utils.Discard
func DiscardLogging() (restore func()) {
	std := logrus.StandardLogger()
	prevOut, prevLevel := std.Out, std.Level
	restore = func() {
		logrus.SetOutput(prevOut)
		logrus.SetLevel(prevLevel)
	}

	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(logrus.FatalLevel)
	return restore
}
