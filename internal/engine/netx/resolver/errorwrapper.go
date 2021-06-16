package resolver

import (
	"context"
	"errors"

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
		Failure:       ClassifyResolveFailure(err),
		Operation:     errorx.ResolveOperation,
		TransactionID: txID,
	}.MaybeBuild()
	return addrs, err
}

// ErrDNSBogon indicates that we found a bogon address. This is the
// correct value with which to initialize MeasurementRoot.ErrDNSBogon
// to tell this library to return an error when a bogon is found.
var ErrDNSBogon = errors.New("dns: detected bogon address")

func ClassifyResolveFailure(err error) string {
	if errors.Is(err, ErrDNSBogon) {
		return errorx.FailureDNSBogonError // not in MK
	}
	return ""
}

var _ Resolver = ErrorWrapperResolver{}
