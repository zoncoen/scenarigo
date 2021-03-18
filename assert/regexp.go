package assert

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

var typeString = reflect.TypeOf("")

// Regexp returns an assertion to ensure a value matches the regular expression pattern.
func Regexp(expr string) Assertion {
	pattern, err := regexp.Compile(expr)
	if err != nil {
		return AssertionFunc(func(v interface{}) error {
			return err
		})
	}
	return AssertionFunc(func(v interface{}) error {
		if s, ok := v.(string); ok {
			if pattern.MatchString(s) {
				return nil
			}
			return fmt.Errorf(`does not match the pattern "%s"`, expr)
		}

		converted, err := convert(v, typeString)
		if err == nil {
			if s, ok := converted.(string); ok {
				if pattern.MatchString(s) {
					return nil
				}
				return fmt.Errorf(`does not match the pattern "%s"`, expr)
			}
		}

		return errors.New("expect string")
	})
}
