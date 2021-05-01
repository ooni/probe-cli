// Package transactionid contains code to share the transactionID
package transactionid

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

type contextkey struct{}

var id = atomicx.NewInt64()

// WithTransactionID returns a copy of ctx with TransactionID
func WithTransactionID(ctx context.Context) context.Context {
	return context.WithValue(
		ctx, contextkey{}, id.Add(1),
	)
}

// ContextTransactionID returns the TransactionID of the context, or zero
func ContextTransactionID(ctx context.Context) int64 {
	id, _ := ctx.Value(contextkey{}).(int64)
	return id
}
