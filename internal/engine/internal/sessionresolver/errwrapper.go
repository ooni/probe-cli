package sessionresolver

//
// Error wrapping
//

import (
	"errors"
	"fmt"
)

// errWrapper wraps an error to include the URL of the
// resolver that we're currently using.
type errWrapper struct {
	err error
	url string
}

// newErrWrapper creates a new err wrapper.
func newErrWrapper(err error, URL string) *errWrapper {
	return &errWrapper{
		err: err,
		url: URL,
	}
}

// Error implements error.Error.
func (ew *errWrapper) Error() string {
	return fmt.Sprintf("<%s> %s", ew.url, ew.err.Error())
}

// Is allows consumers to query for the type of the underlying error.
func (ew *errWrapper) Is(target error) bool {
	return errors.Is(ew.err, target)
}

// Unwrap returns the underlying error.
func (ew *errWrapper) Unwrap() error {
	return ew.err
}
