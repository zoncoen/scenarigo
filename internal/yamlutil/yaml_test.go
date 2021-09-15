package yamlutil

import (
	"testing"
)

func TestRawMessage_UnmarshalYAML(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expect := "hello"
		var msg RawMessage
		if err := msg.UnmarshalYAML([]byte(expect)); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}
		if got := string(msg); got != expect {
			t.Errorf("expect %q but got %q", expect, got)
		}
	})
	t.Run("nil", func(t *testing.T) {
		var msg *RawMessage
		if err := msg.UnmarshalYAML([]byte("'hello")); err == nil {
			t.Error("no error")
		}
	})
}

func TestRawMessage_Unmarshal(t *testing.T) {
	expect := "hello"
	var msg RawMessage
	if err := msg.UnmarshalYAML([]byte(expect)); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	var got string
	if err := msg.Unmarshal(&got); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if got != expect {
		t.Errorf("expect %q but got %q", expect, got)
	}
}
