package measurex

// NewFailure creates an archival failure from an error. We
// cannot round trip an error using JSON, so we serialize to this
// intermediate format that is a sort of Optional<string>.
func NewFailure(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
