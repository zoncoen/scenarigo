package marshaler

import (
	yamljson "github.com/kubernetes-sigs/yaml"
	"github.com/zoncoen/yaml"
)

type jsonMarshaler struct{}

// MediaType implements RequestMarshaler interface.
func (m *jsonMarshaler) MediaType() string {
	return "application/json"
}

// Marshal implements RequestMarshaler interface.
func (m *jsonMarshaler) Marshal(v interface{}) ([]byte, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	jb, err := yamljson.YAMLToJSON(b)
	if err != nil {
		return nil, err
	}
	return jb, nil
}
