package pdsl

// Filter is a function that transforms a stream of A into a stream of B.
type Filter[A, B any] func(inputs <-chan A) <-chan B
