package resolver

import (
	"context"
	"errors"
	"strings"

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
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrDNSBogon) {
		return errorx.FailureDNSBogonError // not in MK
	}
	if strings.HasSuffix(err.Error(), "no such host") {
		// This is dns_lookup_error in MK but such error is used as a
		// generic "hey, the lookup failed" error. Instead, this error
		// that we return here is significantly more specific.
		return errorx.FailureDNSNXDOMAINError
	}
	return ""
}

var _ Resolver = ErrorWrapperResolver{}
