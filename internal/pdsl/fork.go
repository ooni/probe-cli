package pdsl

// Fork executes N copies of a [Filter] and returns the list of channels to [Merge]. As a
// special case, when N is less than one, this function returns a single channel.
func Fork[A, B any](N int, f Filter[A, B], inputs <-chan A) (outputs []<-chan B) {
	switch {
	case N <= 1:
		outputs = append(outputs, f(inputs))
	default:
		for idx := 0; idx < N; idx++ {
			outputs = append(outputs, f(inputs))
		}
	}
	return
}
