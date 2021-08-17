package webconnectivity

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
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
	config.Out <- CtrlDNSResult{Failure: newfailure(err), Addrs: addrs}
}
