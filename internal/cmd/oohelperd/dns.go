package main

//
// DNS measurements
//

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// newfailure is a convenience shortcut to save typing
var newfailure = tracex.NewFailure

// ctrlDNSResult is the result of the DNS check performed by
// the Web Connectivity test helper.
type ctrlDNSResult = model.THDNSResult

// dnsConfig configures the DNS check.
type dnsConfig struct {
	// Domain is the MANDATORY domain to resolve.
	Domain string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NewResolver is the MANDATORY factory to create a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// Out is the channel where we publish the results.
	Out chan ctrlDNSResult

	// Wg allows to synchronize with the parent.
	Wg *sync.WaitGroup
}

// dnsDo performs the DNS check.
func dnsDo(ctx context.Context, config *dnsConfig) {
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer config.Wg.Done()
	reso := config.NewResolver(config.Logger)
	defer reso.CloseIdleConnections()
	ol := measurexlite.NewOperationLogger(config.Logger, "DNSLookup %s", config.Domain)
	addrs, err := reso.LookupHost(ctx, config.Domain)
	ol.Stop(err)
	if addrs == nil {
		addrs = []string{} // fix: the old test helper did that
	}
	failure := dnsMapFailure(newfailure(err))
	config.Out <- ctrlDNSResult{
		Failure: failure,
		Addrs:   addrs,
		ASNs:    []int64{}, // unused by the TH and not serialized
	}
}

// dnsMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L430
func dnsMapFailure(failure *string) *string {
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureDNSNXDOMAINError:
			// We have a name for this string because dnsanalysis.go is
			// already checking for this specific error string.
			s := model.THDNSNameError
			return &s
		case netxlite.FailureDNSNoAnswer:
			// In this case the legacy TH would produce an empty
			// reply that is not attached to any error.
			//
			// See https://github.com/ooni/probe/issues/1707#issuecomment-944322725
			return nil
		case netxlite.FailureDNSNonRecoverableFailure,
			netxlite.FailureDNSRefusedError,
			netxlite.FailureDNSServerMisbehaving,
			netxlite.FailureDNSTemporaryFailure:
			s := "dns_server_failure"
			return &s
		default:
			s := "unknown_error"
			return &s
		}
	}
}
