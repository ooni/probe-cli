package resolver

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/transactionid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// ErrorWrapperResolver is a Resolver that knows about wrapping errors.
type ErrorWrapperResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r ErrorWrapperResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	dialID := dialid.ContextDialID(ctx)
	txID := transactionid.ContextTransactionID(ctx)
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	err = errorx.SafeErrWrapperBuilder{
		DialID:        dialID,
		Error:         err,
		Operation:     errorx.ResolveOperation,
		TransactionID: txID,
	}.MaybeBuild()
	return addrs, err
}

var _ Resolver = ErrorWrapperResolver{}
