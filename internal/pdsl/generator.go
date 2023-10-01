package pdsl

// Generator is a function that takes A in input and generates a stream of B in output.
type Generator[A, B any] func(input A) <-chan B
