package unmarshaler

import (
	"reflect"
	"testing"
)

func TestBinaryUnmarshaler_Unmarshal(t *testing.T) {
	var um binaryUnmarshaler
	var i interface{}
	d := []byte("test")
	if err := um.Unmarshal(d, &i); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, i) {
		t.Errorf(`expected "%#v" but got "%#v"`, d, i)
	}
}
