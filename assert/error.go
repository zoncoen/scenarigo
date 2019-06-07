package assert

import "github.com/hashicorp/go-multierror"

// Error is an error type to track multiple errors.
type Error = multierror.Error

// AppendError is a helper function that will append more errors
// onto an Error in order to create a larger multi-error.
var AppendError = multierror.Append
