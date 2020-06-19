package reporter

import "testing"

func TestFromT(t *testing.T) {
	var _ Reporter = FromT(t)
}

func Test_TestingLog(t *testing.T) {
	rptr := FromT(t)
	rptr.Log("log test")
	rptr.Run("subtest", func(rptr Reporter) {
		rptr.Logf("%s", "log test")
		rptr.Skip()
	})
}
