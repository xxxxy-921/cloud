package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONText is a json.RawMessage wrapper that works with both SQLite ([]byte)
// and PostgreSQL (string) drivers. Use instead of json.RawMessage in GORM models.
type JSONText json.RawMessage

func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONText) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = JSONText(v)
	default:
		return fmt.Errorf("JSONText.Scan: unsupported type %T", value)
	}
	return nil
}

func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JSONText) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONText.UnmarshalJSON: nil pointer")
	}
	*j = append((*j)[:0], data...)
	return nil
}
