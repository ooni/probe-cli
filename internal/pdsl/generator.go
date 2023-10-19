package pdsl

// Generator is a function that takes A in input and generates a stream of B in output.
type Generator[A, B any] func(input A) <-chan Result[B]

type generatorOperation[A, B any] func(input A) ([]B, error)

func startGeneratorService[A, B any](op generatorOperation[A, B]) Generator[A, B] {
	return func(input A) <-chan Result[B] {
		// create the outputs channel
		outputs := make(chan Result[B])

		go func() {
			// make sure we close channel when done
			defer close(outputs)

			// invoke the operation
			results, err := op(input)

			// handle failure
			if err != nil {
				outputs <- NewResultError[B](err)
				return
			}

			// handle success
			for _, result := range results {
				outputs <- NewResultValue(result)
			}
		}()

		return outputs
	}
}
