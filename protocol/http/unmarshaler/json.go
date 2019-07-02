package unmarshaler

import (
	"encoding/json"
)

func init() {
	Register(&jsonUnmarshaler{})
}

type jsonUnmarshaler struct{}

// MediaType implements ResponseUnmarshaler interface.
func (um *jsonUnmarshaler) MediaType() string {
	return "application/json"
}

// Unmarshal implements ResponseUnmarshaler interface.
func (um *jsonUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, &v)
}
