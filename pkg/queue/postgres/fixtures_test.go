package postgres

import (
	"os"
	"testing"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/goleak"
)

var (
	queueID1 = uuid.NewV4().String()
	queueID2 = uuid.NewV4().String()
	queueID3 = uuid.NewV4().String()
	progress = []byte(`{"scale": 99}`)
	spec     = []byte(`{"field": "value"}`)
)

func verifyLeak(t *testing.T) {
	_, exists := os.LookupEnv("CI")
	if exists {
		return
	}
	goleak.VerifyNone(t)
}
