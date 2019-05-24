package reporter

import "testing"

func TestFromT(t *testing.T) {
	var _ Reporter = FromT(t)
}
