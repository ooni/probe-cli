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

// newfailure is a convenience shortcut to save typing.
var newfailure = tracex.NewFailure

// ctrlDNSResult is the result returned by the [dnsDo] function and
// included into the response sent to the client.
type ctrlDNSResult = model.THDNSResult

// dnsConfig contains configuration for the [dnsDo] function.
type dnsConfig struct {
	// Domain is the MANDATORY domain to resolve.
	Domain string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NewResolver is the MANDATORY factory to create a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// Out is the MANDATORY channel where we publish the results.
	Out chan ctrlDNSResult

	// Wg is MANDATORY and allows [dnsDo] to synchronize with the caller.
	Wg *sync.WaitGroup
}

// dnsDo performs a DNS micro-measurement using the given [dnsConfig].
func dnsDo(ctx context.Context, config *dnsConfig) {
	// make sure this micro-measurement is bounded in time
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// make sure the caller knows when we're done
	defer config.Wg.Done()

	// create a temporary resolver for this micro-measurement
	reso := config.NewResolver(config.Logger)
	defer reso.CloseIdleConnections()

	// perform and log the actual DNS lookup
	ol := measurexlite.NewOperationLogger(config.Logger, "DNSLookup %s", config.Domain)
	addrs, err := reso.LookupHost(ctx, config.Domain)
	ol.Stop(err)

	// make sure we return an empty slice on failure because this
	// is what the legacy TH would have done.
	if addrs == nil {
		addrs = []string{}
	}

	// map the OONI failure to the failure string that the legacy
	// TH would have returned to us in this case.
	failure := dnsMapFailure(newfailure(err))

	// emit the result; note that the ASNs field is unused by
	// the TH and is not serialized to JSON.
	config.Out <- ctrlDNSResult{
		Failure: failure,
		Addrs:   addrs,
		ASNs:    []int64{},
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
