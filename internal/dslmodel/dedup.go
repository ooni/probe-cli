package dslmodel

import "context"

// Deduplicable is something that we can deduplicate.
type Deduplicable interface {
	DedupKey() string
}

// DedupPipeline deduplicates [Deduplicable] content from a pipeline.
func DedupPipeline[T Deduplicable]() Pipeline[T, T] {
	return PipelineFunc[T, T](func(ctx context.Context, rt Runtime, inputs <-chan Result[T]) <-chan Result[T] {
		outputs := make(chan Result[T])

		go func() {
			defer close(outputs)

			already := make(map[string]bool)
			for input := range inputs {
				if err := input.Err; err != nil {
					outputs <- input
					continue
				}

				key := input.Value.DedupKey()
				if already[key] {
					continue
				}

				already[key] = true
				outputs <- input
			}
		}()

		return outputs
	})
}
