package dslmodel

import "context"

// SyncOperation is a synchronous operation producing a result.
type SyncOperation[A, B any] func(ctx context.Context, rt Runtime, input A) Result[B]

// SyncOperationToFilter converts a [SyncOperation] to a [Filter].
func SyncOperationToFilter[A, B any](operation SyncOperation[A, B]) Filter[A, B] {
	return FilterFunc[A, B](func(ctx context.Context, rt Runtime, minput Result[A]) Result[B] {
		if err := minput.Err; err != nil {
			return NewResultError[B](err)
		}
		return operation(ctx, rt, minput.Value)
	})
}

// AsyncOperation is an asynchronous operation producing a stream of results.
type AsyncOperation[A, B any] func(ctx context.Context, rt Runtime, input A) <-chan Result[B]

// AsyncOperationToGenerator converts a [AsyncOperation] to a [Generator].
func AsyncOperationToGenerator[A, B any](operation AsyncOperation[A, B]) Generator[A, B] {
	return GeneratorFunc[A, B](func(ctx context.Context, rt Runtime, minput Result[A]) <-chan Result[B] {
		if err := minput.Err; err != nil {
			return StreamResultError[B](err)
		}
		return operation(ctx, rt, minput.Value)
	})
}
