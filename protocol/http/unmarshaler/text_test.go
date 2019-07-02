package unmarshaler

import (
	"testing"
)

func TestTextUnUnmarshaler_Unmarshal(t *testing.T) {
	var um textUnmarshaler
	var i interface{}
	s := "test"
	if err := um.Unmarshal([]byte(s), &i); err != nil {
		t.Fatal(err)
	}
	if i != s {
		t.Errorf(`expected "%s" but got %v`, s, i)
	}
}
