package parser

import (
	"strings"
	"testing"

	"github.com/zoncoen/scenarigo/template/token"
)

func TestScanner_Read(t *testing.T) {
	tests := map[string]struct {
		s         string
		buf       []rune
		expect    rune
		expectPos int
	}{
		"EOF": {
			expect:    eof,
			expectPos: 1,
		},
		"read": {
			s:         "abc",
			expect:    'a',
			expectPos: 2,
		},
		"read from buffer": {
			s:         "abc",
			buf:       []rune{'A', 'B', 'C'},
			expect:    'A',
			expectPos: 2,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			s := newScanner(strings.NewReader(test.s))
			s.buf = test.buf
			if got := s.read(); got != test.expect {
				t.Errorf("expected %q but got %q", test.expect, got)
			}
			if got := s.pos; got != test.expectPos {
				t.Errorf("expected %d but got %d", test.expectPos, got)
			}
		})
	}
}

func TestScanner_Unread(t *testing.T) {
	s := &scanner{
		pos: 4,
	}
	s.unread('a')
	s.unread('b')
	s.unread('c')
	if got, expect := string(s.buf), "abc"; got != expect {
		t.Errorf("expected %q but got %q", expect, got)
	}
	if got, expect := s.pos, 1; got != expect {
		t.Errorf("expected %d but got %d", expect, got)
	}
}

func TestScanner_Scan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		type result struct {
			pos int
			tok token.Token
			lit string
		}
		tests := map[string]struct {
			src      string
			expected []result
		}{
			"empty": {
				src:      "",
				expected: []result{},
			},
			"no parameter": {
				src: "test",
				expected: []result{
					{
						pos: 1,
						tok: token.STRING,
						lit: "test",
					},
				},
			},
			"trailing {": {
				src: "test {",
				expected: []result{
					{
						pos: 1,
						tok: token.STRING,
						lit: "test {",
					},
				},
			},
			"trailing {{": {
				src: "test {{",
				expected: []result{
					{
						pos: 1,
						tok: token.STRING,
						lit: "test ",
					},
					{
						pos: 6,
						tok: token.LDBRACE,
						lit: "{{",
					},
				},
			},
			"trailing {{}}": {
				src: "test {{}}",
				expected: []result{
					{
						pos: 1,
						tok: token.STRING,
						lit: "test ",
					},
					{
						pos: 6,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 8,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"just a STRING": {
				src: `{{ "test" }}`,
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 4,
						tok: token.STRING,
						lit: "test",
					},
					{
						pos: 11,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"just a IDENT": {
				src: "{{  test  }}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 5,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 11,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"IDENT with special character": {
				src: "{{a-b_c}}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 3,
						tok: token.IDENT,
						lit: "a-b_c",
					},
					{
						pos: 8,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"parameter with raw string": {
				src: "prefix-{{test}}-suffix",
				expected: []result{
					{
						pos: 1,
						tok: token.STRING,
						lit: "prefix-",
					},
					{
						pos: 8,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 10,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 14,
						tok: token.RDBRACE,
						lit: "}}",
					},
					{
						pos: 16,
						tok: token.STRING,
						lit: "-suffix",
					},
				},
			},
			"IDENT.IDENT.IDENT": {
				src: "{{test.cases.length}}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 3,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 7,
						tok: token.PERIOD,
						lit: ".",
					},
					{
						pos: 8,
						tok: token.IDENT,
						lit: "cases",
					},
					{
						pos: 13,
						tok: token.PERIOD,
						lit: ".",
					},
					{
						pos: 14,
						tok: token.IDENT,
						lit: "length",
					},
					{
						pos: 20,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"IDENT[INT][INT]": {
				src: "{{test[0][100]}}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 3,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 7,
						tok: token.LBRACK,
						lit: "[",
					},
					{
						pos: 8,
						tok: token.INT,
						lit: "0",
					},
					{
						pos: 9,
						tok: token.RBRACK,
						lit: "]",
					},
					{
						pos: 10,
						tok: token.LBRACK,
						lit: "[",
					},
					{
						pos: 11,
						tok: token.INT,
						lit: "100",
					},
					{
						pos: 14,
						tok: token.RBRACK,
						lit: "]",
					},
					{
						pos: 15,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"function call": {
				src: "{{ test() }}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 4,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 8,
						tok: token.LPAREN,
						lit: "(",
					},
					{
						pos: 9,
						tok: token.RPAREN,
						lit: ")",
					},
					{
						pos: 11,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"function call with args": {
				src: "{{ test(1,2,3) }}",
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 4,
						tok: token.IDENT,
						lit: "test",
					},
					{
						pos: 8,
						tok: token.LPAREN,
						lit: "(",
					},
					{
						pos: 9,
						tok: token.INT,
						lit: "1",
					},
					{
						pos: 10,
						tok: token.COMMA,
						lit: ",",
					},
					{
						pos: 11,
						tok: token.INT,
						lit: "2",
					},
					{
						pos: 12,
						tok: token.COMMA,
						lit: ",",
					},
					{
						pos: 13,
						tok: token.INT,
						lit: "3",
					},
					{
						pos: 14,
						tok: token.RPAREN,
						lit: ")",
					},
					{
						pos: 16,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
			"add": {
				src: `{{"test"+"1"}}`,
				expected: []result{
					{
						pos: 1,
						tok: token.LDBRACE,
						lit: "{{",
					},
					{
						pos: 3,
						tok: token.STRING,
						lit: "test",
					},
					{
						pos: 9,
						tok: token.ADD,
						lit: "+",
					},
					{
						pos: 10,
						tok: token.STRING,
						lit: "1",
					},
					{
						pos: 13,
						tok: token.RDBRACE,
						lit: "}}",
					},
				},
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				s := newScanner(strings.NewReader(test.src))
				for i, e := range test.expected {
					pos, tok, lit := s.scan()
					if tok == token.EOF {
						t.Fatalf("[%d] unexpected EOF", i)
					}
					if got, expected := pos, e.pos; got != expected {
						t.Fatalf(`[%d] expected %d but got %d`, i, expected, got)
					}
					if got, expected := tok, e.tok; got != expected {
						t.Fatalf(`[%d] expected "%s" but got "%s"`, i, expected, got)
					}
					if got, expected := lit, e.lit; got != expected {
						t.Fatalf(`[%d] expected "%s" but got "%s"`, i, expected, got)
					}
				}
				pos, tok, lit := s.scan()
				if tok != token.EOF {
					t.Fatalf(`expected EOF but got %d:%s:%s`, pos, tok, lit)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			src string
			pos int
			lit string
		}{
			"invalid identifier": {
				src: "{{_a}}",
				pos: 3,
				lit: "_",
			},
			"invalid integer index": {
				src: "{{01}}",
				pos: 3,
				lit: "01",
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				s := newScanner(strings.NewReader(test.src))
				for {
					pos, tok, lit := s.scan()
					if tok == token.EOF {
						t.Fatal("unexpected EOF")
					}
					if tok == token.ILLEGAL {
						if got, expected := pos, test.pos; got != expected {
							t.Errorf(`expected %d but got %d`, expected, got)
						}
						if got, expected := lit, test.lit; got != expected {
							t.Errorf(`expected "%s" but got "%s"`, expected, got)
						}
						break
					}
				}
			})
		}
	})
}
