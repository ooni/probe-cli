package dslmodel

import "context"

// FunctionWithScalarResult is a function that takes A in input and returns B or an error.
type FunctionWithScalarResult[A, B any] func(ctx context.Context, rt Runtime, input A) (B, error)

// FunctionWithScalarResultToSyncOperation converts a [FunctionWithScalarResult] to a [SyncOperation].
func FunctionWithScalarResultToSyncOperation[A, B any](f FunctionWithScalarResult[A, B]) SyncOperation[A, B] {
	return func(ctx context.Context, rt Runtime, input A) Result[B] {
		result, err := f(ctx, rt, input)
		if err != nil {
			return NewResultError[B](err)
		}
		return NewResultValue(result)
	}
}

// FunctionWithSliceResult is a function that takes A in input and returns a slice of B or an error.
type FunctionWithSliceResult[A, B any] func(ctx context.Context, rt Runtime, input A) ([]B, error)

// FunctionWithSliceResultToAsyncOperation converts a [FunctionWithSliceResult] to a [AsyncOperation].
func FunctionWithSliceResultToAsyncOperation[A, B any](f FunctionWithSliceResult[A, B]) AsyncOperation[A, B] {
	return func(ctx context.Context, rt Runtime, input A) <-chan Result[B] {
		outs, err := f(ctx, rt, input)
		if err != nil {
			return StreamResultError[B](err)
		}
		return StreamResultValue(outs...)
	}
}
