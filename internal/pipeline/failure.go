package pipeline

// Failure is a failure string. The empty string represents the absence of failure.
type Failure string

// NewFailure constructs a new [Failure] instance.
func NewFailure(in *string) (out Failure) {
	if in != nil {
		out = Failure(*in)
	}
	return
}
