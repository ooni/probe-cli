package resolver

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// ErrorWrapperResolver is a Resolver that knows about wrapping errors.
type ErrorWrapperResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r ErrorWrapperResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	err = errorsx.SafeErrWrapperBuilder{
		Classifier: errorsx.ClassifyResolveFailure,
		Error:      err,
		Operation:  errorsx.ResolveOperation,
	}.MaybeBuild()
	return addrs, err
}

var _ Resolver = ErrorWrapperResolver{}
