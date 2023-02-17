package tortunnel

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Tunnel is a [model.Tunnel] implemented using tor.
type Tunnel struct {
	// bootstrapTime is the duration of the bootstrap.
	bootstrapTime time.Duration

	// instance is the running tor instance.
	instance *tor.Tor

	// maybeDeleteTunnelDir is a cleanup function that deletes
	// the tunnel dir if it's a temporary directory.
	maybeDeleteTunnelDir func()

	// name is the tunnel name.
	name string

	// proxy is the SOCKS5 proxy URL.
	proxy *url.URL

	// stopOnce allows us to call stop just once.
	stopOnce sync.Once
}

var _ model.Tunnel = &Tunnel{}

// BootstrapTime implements model.Tunnel
func (t *Tunnel) BootstrapTime() time.Duration {
	return t.bootstrapTime
}

// LookupProbeIP implements model.Tunnel
func (t *Tunnel) LookupProbeIP(ctx context.Context) (string, error) {
	panic("not implemented")
}

// Name implements model.Tunnel
func (t *Tunnel) Name() string {
	return t.name
}

// NewDNSOverHTTPSResolver implements model.Tunnel
func (t *Tunnel) NewDNSOverHTTPSResolver(logger model.Logger, URL string) model.Resolver {
	httptxp := t.NewHTTPTransport(logger)
	dnsxp := netxlite.NewDNSOverHTTPSTransportWithHTTPTransport(httptxp, URL)
	return netxlite.WrapResolver(logger, netxlite.NewUnwrappedParallelResolver(dnsxp))
}

// NewHTTPTransport implements model.Tunnel
func (t *Tunnel) NewHTTPTransport(logger model.Logger) model.HTTPTransport {
	dialer := netxlite.NewDialerWithoutResolver(logger)
	dialer = netxlite.MaybeWrapWithProxyDialer(dialer, t.proxy)
	tlsDialer := netxlite.NewTLSDialer(dialer, netxlite.NewTLSHandshakerStdlib(logger))
	return netxlite.NewHTTPTransport(logger, dialer, tlsDialer)
}

// SOCKS5ProxyURL implements model.Tunnel
func (t *Tunnel) SOCKS5ProxyURL() *url.URL {
	return t.proxy
}

// Stop implements model.Tunnel
func (t *Tunnel) Stop() {
	t.stopOnce.Do(func() {
		t.instance.Close()
		t.maybeDeleteTunnelDir()
	})
}
