package measurexlite

// NewFailure creates an OONI failure from an error. In github.com/ooni/spec
// we define an OONI failure as a nullable string.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md
func NewFailure(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
