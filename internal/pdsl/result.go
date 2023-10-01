package pdsl

// Result contains either a value or an error.
type Result[T any] struct {
	Err   error
	Value T
}

// NewResultValue contains a result containing a value.
func NewResultValue[T any](value T) Result[T] {
	return Result[T]{nil, value}
}

// NewResultError contains a result containing an error.
func NewResultError[T any](err error) Result[T] {
	return Result[T]{err, *new(T)}
}
