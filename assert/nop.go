package assert

// Nop returns an assertion that does not assert anything.
func Nop() Assertion {
	return AssertionFunc(func(v interface{}) error {
		return nil
	})
}
