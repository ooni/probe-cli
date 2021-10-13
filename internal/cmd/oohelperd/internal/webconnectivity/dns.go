package webconnectivity

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newfailure is a convenience shortcut to save typing
var newfailure = archival.NewFailure

// CtrlDNSResult is the result of the DNS check performed by
// the Web Connectivity test helper.
type CtrlDNSResult = webconnectivity.ControlDNSResult

// DNSConfig configures the DNS check.
type DNSConfig struct {
	Domain   string
	Out      chan CtrlDNSResult
	Resolver netx.Resolver
	Wg       *sync.WaitGroup
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, config *DNSConfig) {
	defer config.Wg.Done()
	addrs, err := config.Resolver.LookupHost(ctx, config.Domain)
	if addrs == nil {
		addrs = []string{} // fix: the old test helper did that
	}
	failure := dnsMapFailure(newfailure(err))
	config.Out <- CtrlDNSResult{Failure: failure, Addrs: addrs}
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
			// alreayd checking for this specific error string.
			s := webconnectivity.DNSNameError
			return &s
		case netxlite.FailureDNSNoAnswer,
			netxlite.FailureDNSNonRecoverableFailure,
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
