package reporter

import "testing"

func TestMatcher_Match(t *testing.T) {
	tests := map[string]struct {
		name string
		run  string
		ok   []string
		ng   []string
	}{
		"empty -run": {
			name: "foo/bar",
			run:  "",
			ok:   []string{"hoge", "fuga"},
		},
		"-run is a part of the test name": {
			name: "foo/bar/baz",
			run:  "foo/bar",
			ok:   []string{"hoge"}, // match all
		},
		"-run is the same as test name": {
			name: "foo/bar",
			run:  "foo/bar",
			ok:   []string{"hoge"}, // match all
		},
		"-run includes the test name": {
			name: "foo",
			run:  "foo/bar",
			ok:   []string{"bar", "baz"},
			ng:   []string{"hoge"},
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			m, err := newMatcher(test.name, test.run)
			if err != nil {
				t.Fatalf("failed to create matcher: %s", err)
			}
			if test.ok != nil {
				for i, s := range test.ok {
					if !m.match(s, i+1) {
						t.Fatalf("%d: %s does not match", i, s)
					}
				}
			}
			if i := len(test.ng) - 1; i >= 0 {
				if s := test.ng[i]; m.match(s, i+1) {
					t.Fatalf("%d: %s match", i, s)
				}
			}
		})
	}

	t.Run("compile error", func(t *testing.T) {
		if _, err := newMatcher("TestFoo", "TestFoo/[a-z"); err == nil {
			t.Fatal("no error")
		}
	})
}
