package resolver

import (
	"context"
)

// ErrorWrapperResolver is a Resolver that knows about wrapping errors.
type ErrorWrapperResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r ErrorWrapperResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	if err != nil {
		err = NewErrResolve(&err)
	}
	return addrs, err
}

type ErrResolve struct {
	error
}

func NewErrResolve(e *error) *ErrResolve {
	return &ErrResolve{*e}
}

func (e *ErrResolve) Unwrap() error {
	return e.error
}

var _ Resolver = ErrorWrapperResolver{}
