package pdsl

// Filter is a function that transforms a stream of [Result] A into a stream of [Result] B.
type Filter[A, B any] func(inputs <-chan Result[A]) <-chan Result[B]
