package iplookup

//
// Wrapper to ensure resolved IP addresses match the expected family.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// familyWrapperLookup ensures IP addresses match the expected family.
type familyWrapperLookup struct {
	child fallback.Service[model.AddressFamily, string]
}

var _ fallback.Service[model.AddressFamily, string] = &familyWrapperLookup{}

// newFamilyWrapperLookup creates a new [familyWrapperLookup] instance.
func newFamilyWrapperLookup(child fallback.Service[model.AddressFamily, string]) *familyWrapperLookup {
	return &familyWrapperLookup{child}
}

// Run implements fallback.Service
func (svc *familyWrapperLookup) Run(ctx context.Context, family model.AddressFamily) (string, error) {
	addr, err := svc.child.Run(ctx, family)
	if err != nil {
		return "", err
	}
	if !netxlite.AddressBelongsToAddressFamily(addr, family) {
		return "", ErrInvalidIPAddressForFamily
	}
	return addr, nil
}

// URL implements fallback.Service
func (svc *familyWrapperLookup) URL() string {
	return svc.child.URL()
}
