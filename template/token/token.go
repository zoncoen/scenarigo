// Package token defines constants representing the lexical tokens.
package token

type Token int

const (
	ILLEGAL Token = iota
	EOF

	STRING // "text"
	INT    // 123
	IDENT  // vars

	ADD // +

	LPAREN  // (
	RPAREN  // )
	LBRACK  // [
	RBRACK  // ]
	LDBRACE // {{
	RDBRACE // }}
	COMMA   // ,
	PERIOD  // .
)

// String returns t as string.
func (t Token) String() string {
	switch t {
	case EOF:
		return "EOF"
	case STRING:
		return "string"
	case INT:
		return "int"
	case IDENT:
		return "ident"
	case ADD:
		return "add"
	case LPAREN:
		return "lparen"
	case RPAREN:
		return "rparen"
	case LBRACK:
		return "lbrack"
	case RBRACK:
		return "rbrack"
	case LDBRACE:
		return "ldbrace"
	case RDBRACE:
		return "rdbrace"
	case COMMA:
		return "comma"
	case PERIOD:
		return "period"
	}
	return "illegal"
}
