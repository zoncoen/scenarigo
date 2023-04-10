// Package token defines constants representing the lexical tokens.
package token

type Token int

const (
	ILLEGAL Token = iota
	EOF

	STRING // "text"
	INT    // 123
	FLOAT  // 1.23
	BOOL   // true
	IDENT  // vars

	ADD  // +
	SUB  // -
	MUL  // *
	QUO  // /
	REM  // %
	CALL // }}:\n

	LPAREN    // (
	RPAREN    // )
	LBRACK    // [
	RBRACK    // ]
	LDBRACE   // {{
	RDBRACE   // }}
	COMMA     // ,
	PERIOD    // .
	LARROW    // <-
	LINEBREAK // end of a larrow expression argument
)

// String returns t as string.
func (t Token) String() string {
	switch t {
	case ILLEGAL:
		return "illegal"
	case EOF:
		return "EOF"
	case STRING:
		return "string"
	case INT:
		return "int"
	case FLOAT:
		return "float"
	case BOOL:
		return "bool"
	case IDENT:
		return "ident"
	case ADD:
		return "+"
	case SUB:
		return "-"
	case MUL:
		return "*"
	case QUO:
		return "/"
	case REM:
		return "%"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACK:
		return "["
	case RBRACK:
		return "]"
	case LDBRACE:
		return "{{"
	case RDBRACE:
		return "}}"
	case COMMA:
		return ","
	case PERIOD:
		return "."
	case LARROW:
		return "<-"
	case LINEBREAK:
		return "line break"
	default:
		return "unknown"
	}
}

// A set of constants for precedence-based expression parsing.
// Non-operators have lowest precedence.
const (
	LowestPrec  = 0 // non-operators
	HighestPrec = 3
)

// Precedence returns the operator precedence of the binary
// operator op. If op is not a binary operator, the result
// is LowestPrecedence.
func (t Token) Precedence() int {
	switch t {
	case ADD, SUB, LARROW, LDBRACE, STRING:
		return 1
	case MUL, QUO, REM:
		return 2
	default:
		return LowestPrec
	}
}
