package sessionresolver

import "fmt"

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
