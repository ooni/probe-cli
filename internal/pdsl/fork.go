package pdsl

// Fork executes N copies of a [Filter] and returns channels to [Merge].
//
// If N is <= 1, [Fork] returns a list containing a single channel.
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
