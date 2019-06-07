package parser

import (
	"fmt"
)

// Errors represents parse errors.
type Errors []*Error

// Append appends a parse error to errs.
func (errs *Errors) Append(pos int, msg string) {
	*errs = append(*errs, &Error{
		pos: pos,
		msg: msg,
	})
}

// Error returns error string.
func (errs Errors) Error() string {
	switch len(errs) {
	case 0:
		return "no errors"
	case 1:
		return errs[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", errs[0], len(errs)-1)
}

// Err returns an error equivalent to this errors.
// If the list is empty, Err returns nil.
func (errs Errors) Err() error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Error represents a parse error.
type Error struct {
	pos int
	msg string
}

// Error returns error string.
func (e *Error) Error() string {
	return fmt.Sprintf("col %d: %s", e.pos, e.msg)
}
