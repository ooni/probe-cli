package sessionresolver

import (
	"errors"
	"fmt"
)

// errwrapper wraps an error to include the URL of the
// resolver that we're currently using.
type errwrapper struct {
	error
	URL string
}

// Error implements error.Error.
func (ew *errwrapper) Error() string {
	return fmt.Sprintf("<%s> %s", ew.URL, ew.error.Error())
}

// Is allows consumers to query for the type of the underlying error.
func (ew *errwrapper) Is(target error) bool {
	return errors.Is(ew.error, target)
}
