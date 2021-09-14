package testutil

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zoncoen/scenarigo/version"
)

func TestReplaceOutput(t *testing.T) {
	str := fmt.Sprintf(`
=== RUN   test.yaml
--- FAIL: test.yaml (1.23s)
    [0] send request
    request:
      method: GET
      url: http://[::]:35233/echo
      header:
        User-Agent:
        - scenarigo/%s
    elapsed time: 0.123456 sec
       6 |     method: GET
       7 |     url: "http://{{env.TEST_HTTP_ADDR}}/echo"
       8 |   expect:
    >  9 |     code: OK
                     ^
    expected OK but got Internal Server Error
FAIL
FAIL    test.yaml      1.234s
FAIL
`, version.String())
	expect := `
=== RUN   test.yaml
--- FAIL: test.yaml (0.00s)
    [0] send request
    request:
      method: GET
      url: http://[::]:12345/echo
      header:
        User-Agent:
        - scenarigo/v1.0.0
    elapsed time: 0.000000 sec
       6 |     method: GET
       7 |     url: "http://{{env.TEST_HTTP_ADDR}}/echo"
       8 |   expect:
    >  9 |     code: OK
                     ^
    expected OK but got Internal Server Error
FAIL
FAIL    test.yaml      0.000s
FAIL
`
	if diff := cmp.Diff(expect, ReplaceOutput(str)); diff != "" {
		t.Errorf("differs (-want +got):\n%s", diff)
	}
}
