package pdsl

// Stream streams a list of results into a channel closed when done.
func Stream[T any](inputs ...Result[T]) <-chan Result[T] {
	out := make(chan Result[T], len(inputs))
	for _, input := range inputs {
		out <- input
	}
	close(out)
	return out
}
