package dialid

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

type contextkey struct{}

var id = atomicx.NewInt64()

// WithDialID returns a copy of ctx with DialID
func WithDialID(ctx context.Context) context.Context {
	return context.WithValue(
		ctx, contextkey{}, id.Add(1),
	)
}

// ContextDialID returns the DialID of the context, or zero
func ContextDialID(ctx context.Context) int64 {
	id, _ := ctx.Value(contextkey{}).(int64)
	return id
}
