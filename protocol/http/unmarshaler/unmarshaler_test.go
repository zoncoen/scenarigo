package unmarshaler

import "testing"

func TestGet(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		um := Get("image/jpeg")
		_, ok := um.(*binaryUnmarshaler)
		if !ok {
			t.Errorf("expected *binaryUnmarshaler but got %T", um)
		}
	})
}
