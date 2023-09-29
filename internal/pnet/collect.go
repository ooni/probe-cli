package pnet

// Collect drains a channel and returns its results.
func Collect[T any](source <-chan T) (sink []T) {
	for entry := range source {
		sink = append(sink, entry)
	}
	return
}
