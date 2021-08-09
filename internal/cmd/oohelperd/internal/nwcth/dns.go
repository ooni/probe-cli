package nwcth

import (
	"context"
	"net"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
)

// TODO(bassosimone,kelmenhorst): figure out if we can _avoid_ using netx here.

// newfailure is a convenience shortcut to save typing
var newfailure = archival.NewFailure

// DNSMeasurement is the result of the DNS check performed by
// the Web Connectivity test helper.
type DNSMeasurement = nwebconnectivity.ControlDNSMeasurement

// DNSConfig configures the DNS check.
type DNSConfig struct {
	Domain string
}

// newResolver creates a new DNS resolver instance
func newResolver() netx.Resolver {
	return netx.NewResolver(netx.Config{Logger: log.Log})
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, config *DNSConfig) *DNSMeasurement {
	if net.ParseIP(config.Domain) != nil {
		// handle IP address format input
		return &DNSMeasurement{Failure: nil, Addrs: []string{config.Domain}}
	}
	resolver := newResolver()
	addrs, err := resolver.LookupHost(ctx, config.Domain)
	return &DNSMeasurement{Failure: newfailure(err), Addrs: addrs}
}
