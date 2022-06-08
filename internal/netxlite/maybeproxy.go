package netxlite

//
// Optional proxy support
//

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/proxy"
)

// proxyDialer is a dialer using a proxy.
type proxyDialer struct {
	Dialer   model.Dialer
	ProxyURL *url.URL
}

// MaybeWrapWithProxyDialer returns the original dialer if the proxyURL is nil
// and otherwise returns a wrapped dialer that implements proxying.
func MaybeWrapWithProxyDialer(dialer model.Dialer, proxyURL *url.URL) model.Dialer {
	if proxyURL == nil {
		return dialer
	}
	return &proxyDialer{
		Dialer:   dialer,
		ProxyURL: proxyURL,
	}
}

var _ model.Dialer = &proxyDialer{}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *proxyDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// ErrProxyUnsupportedScheme indicates we don't support the proxy scheme.
var ErrProxyUnsupportedScheme = errors.New("proxy: unsupported scheme")

// DialContext implements Dialer.DialContext.
func (d *proxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	url := d.ProxyURL
	if url.Scheme != "socks5" {
		return nil, ErrProxyUnsupportedScheme
	}
	// the code at proxy/socks5.go never fails; see https://git.io/JfJ4g
	child, _ := proxy.SOCKS5(network, url.Host, nil, &proxyDialerWrapper{d.Dialer})
	return d.dial(ctx, child, network, address)
}

func (d *proxyDialer) dial(
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
