package testutil

import (
	"fmt"
	"regexp"

	"github.com/zoncoen/scenarigo/version"
)

var (
	dddPattern         = regexp.MustCompile(`\d\.\d\ds`)
	ddddPattern        = regexp.MustCompile(`\d\.\d\d\ds`)
	elapsedTimePattern = regexp.MustCompile(`elapsed time: .+`)
	addrPattern        = regexp.MustCompile(`\[::\]:\d+`)
	userAgentPattern   = regexp.MustCompile(fmt.Sprintf(`- scenarigo/%s`, version.String()))
)

// ReplaceOutput replaces result output.
func ReplaceOutput(s string) string {
	for _, f := range []func(string) string{
		ResetDuration,
		ReplaceAddr,
		ReplaceUserAgent,
	} {
		s = f(s)
	}
	return s
}

// ResetDuration resets durations from result output.
func ResetDuration(s string) string {
	s = dddPattern.ReplaceAllString(s, "0.00s")
	s = ddddPattern.ReplaceAllString(s, "0.000s")
	return elapsedTimePattern.ReplaceAllString(s, "elapsed time: 0.000000 sec")
}

// ReplaceAddr replaces addresses on result output.
func ReplaceAddr(s string) string {
	return addrPattern.ReplaceAllString(s, "[::]:12345")
}

// ReplaceUserAgent replaces User-Agent header on result output.
func ReplaceUserAgent(s string) string {
	return userAgentPattern.ReplaceAllString(s, "- scenarigo/v1.0.0")
}
