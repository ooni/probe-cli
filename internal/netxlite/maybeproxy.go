package netxlite

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/proxy"
)

// MaybeProxyDialer is a dialer that may use a proxy. If the ProxyURL is not configured,
// this dialer is a passthrough for the next Dialer in chain. Otherwise, it will internally
// create a SOCKS5 dialer that will connect to the proxy using the underlying Dialer.
type MaybeProxyDialer struct {
	Dialer   model.Dialer
	ProxyURL *url.URL
}

// NewMaybeProxyDialer creates a new NewMaybeProxyDialer.
func NewMaybeProxyDialer(dialer model.Dialer, proxyURL *url.URL) *MaybeProxyDialer {
	return &MaybeProxyDialer{
		Dialer:   dialer,
		ProxyURL: proxyURL,
	}
}

var _ model.Dialer = &MaybeProxyDialer{}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *MaybeProxyDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// ErrProxyUnsupportedScheme indicates we don't support a protocol scheme.
var ErrProxyUnsupportedScheme = errors.New("proxy: unsupported scheme")

// DialContext implements Dialer.DialContext.
func (d *MaybeProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	url := d.ProxyURL
	if url == nil {
		return d.Dialer.DialContext(ctx, network, address)
	}
	if url.Scheme != "socks5" {
		return nil, ErrProxyUnsupportedScheme
	}
	// the code at proxy/socks5.go never fails; see https://git.io/JfJ4g
	child, _ := proxy.SOCKS5(network, url.Host, nil, &proxyDialerWrapper{d.Dialer})
	return d.dial(ctx, child, network, address)
}

func (d *MaybeProxyDialer) dial(
	ctx context.Context, child proxy.Dialer, network, address string) (net.Conn, error) {
	cd := child.(proxy.ContextDialer) // will work
	return cd.DialContext(ctx, network, address)
}

// proxyDialerWrapper is required because SOCKS5 expects a Dialer.Dial type but internally
// it checks whether DialContext is available and prefers that. So, we need to use this
// structure to cast our inner Dialer the way in which SOCKS5 likes it.
//
// See https://git.io/JfJ4g.
type proxyDialerWrapper struct {
	model.Dialer
}

func (d *proxyDialerWrapper) Dial(network, address string) (net.Conn, error) {
	panic(errors.New("proxyDialerWrapper.Dial should not be called directly"))
}
