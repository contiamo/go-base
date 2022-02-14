package serialization

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONBlob returns an Serializable using json to []byte serialization
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
	switch value := src.(type) {
	case nil:
		return nil
	case []byte:
		return json.Unmarshal(value, b.data)
	case string:
		return json.Unmarshal([]byte(value), b.data)
	default:
		return fmt.Errorf("unknown json object type %T", src)
	}
}

func (b jsonBlob) Value() (driver.Value, error) {
	return json.Marshal(b.data)
}
