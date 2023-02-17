package reporter

import (
	"regexp"
	"strings"
)

type matcher struct {
	patterns []*regexp.Regexp
}

func newMatcher(name, run string) (*matcher, error) {
	if run == "" {
		return nil, nil //nolint: nilnil
	}

	exprs := strings.Split(run, "/")
	c := len(strings.Split(name, "/"))
	if c >= len(exprs) {
		return nil, nil //nolint: nilnil
	}
	exprs = exprs[c:]

	m := &matcher{}
	for _, e := range exprs {
		p, err := regexp.Compile(e)
		if err != nil {
			return nil, err
		}
		m.patterns = append(m.patterns, p)
	}
	return m, nil
}

func (m *matcher) match(s string, depth int) bool {
	if m == nil || depth == 0 {
		return true
	}
	if depth > len(m.patterns) {
		return true
	}
	return m.patterns[depth-1].MatchString(s)
}
