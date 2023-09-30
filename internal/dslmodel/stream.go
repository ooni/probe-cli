package dslmodel

// StreamResultValue streams wraps the given values into a Result
// and stream them to a channel closed when done streaming.
func StreamResultValue[T any](values ...T) <-chan Result[T] {
	outputs := make(chan Result[T], len(values))
	for _, value := range values {
		outputs <- NewResultValue(value)
	}
	close(outputs)
	return outputs
}

// StreamResultError streams wraps the given errs into a Result
// and stream them to a channel closed when done streaming.
func StreamResultError[T any](errs ...error) <-chan Result[T] {
	outputs := make(chan Result[T], len(errs))
	for _, err := range errs {
		outputs <- NewResultError[T](err)
	}
	close(outputs)
	return outputs
}
