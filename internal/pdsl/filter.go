package pdsl

// Filter is a function that transforms a stream of [Result] A into a stream of [Result] B.
type Filter[A, B any] func(inputs <-chan Result[A]) <-chan Result[B]

type filterOperation[A, B any] func(input A) (B, error)

func startFilterService[A, B any](op filterOperation[A, B]) Filter[A, B] {
	return func(minputs <-chan Result[A]) <-chan Result[B] {
		outputs := make(chan Result[B])

		go func() {
			// make sure we close the outputs channel
			defer close(outputs)

			for minput := range minputs {
				// handle the case of previous stage failure
				if err := minput.Err; err != nil {
					outputs <- NewResultError[B](err)
					continue
				}

				// invoke the operation
				result, err := op(minput.Value)

				// handle the error
				if err != nil {
					outputs <- NewResultError[B](err)
					continue
				}

				// handle success
				outputs <- NewResultValue(result)
			}
		}()

		return outputs
	}
}
