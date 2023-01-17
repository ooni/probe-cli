package dslx

//
// Lambda
//

import "context"

// Lambda turns a golang lambda into a Func.
func Lambda[A, B any](fx func(context.Context, A) B) Func[A, B] {
	return &lambdaFunc[A, B]{fx}
}

// lambdaFunc is the type returned by Lambda.
type lambdaFunc[A, B any] struct {
	fun func(context.Context, A) B
}

// Apply implements Func
func (f *lambdaFunc[A, B]) Apply(ctx context.Context, a A) B {
	return f.fun(ctx, a)
}
