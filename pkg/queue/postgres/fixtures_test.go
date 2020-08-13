package postgres

import uuid "github.com/satori/go.uuid"

var (
	queueID1 = uuid.NewV4().String()
	queueID2 = uuid.NewV4().String()
	queueID3 = uuid.NewV4().String()
	progress = []byte(`{"scale": 99}`)
	spec     = []byte(`{"field": "value"}`)
)
