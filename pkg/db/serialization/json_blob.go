package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONBlob returns an Serializable using json serialization
func JSONBlob(value interface{}) Serializable {
	return jsonBlob{data: value}
}

// jsonBlob is a wrapper to present json data to the db
type jsonBlob struct {
	data interface{}
}

func (b jsonBlob) GetData() interface{} {
	return b.data
}

func (b jsonBlob) Scan(src interface{}) error {
	bs, ok := src.([]byte)
	if !ok {
		return errors.New("source must be a byte slice")
	}
	return json.Unmarshal(bs, b.data)
}

func (b jsonBlob) Value() (driver.Value, error) {
	return json.Marshal(b.data)
}
