package dialer

import (
	"context"
	"errors"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

// ProxyDialer is a dialer that uses a proxy. If the ProxyURL is not configured, this
// dialer is a passthrough for the next Dialer in chain. Otherwise, it will internally
// create a SOCKS5 dialer that will connect to the proxy using the underlying Dialer.
type ProxyDialer struct {
	Dialer
	ProxyURL *url.URL
}

// ErrProxyUnsupportedScheme indicates we don't support a protocol scheme.
var ErrProxyUnsupportedScheme = errors.New("proxy: unsupported scheme")

// DialContext implements Dialer.DialContext
func (d ProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	url := d.ProxyURL
	if url == nil {
		return d.Dialer.DialContext(ctx, network, address)
	}
	if url.Scheme != "socks5" {
		return nil, ErrProxyUnsupportedScheme
	}
	// the code at proxy/socks5.go never fails; see https://git.io/JfJ4g
	child, _ := proxy.SOCKS5(
		network, url.Host, nil, proxyDialerWrapper{d.Dialer})
	return d.dial(ctx, child, network, address)
}

func (d ProxyDialer) dial(
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
	Dialer
}

func (d proxyDialerWrapper) Dial(network, address string) (net.Conn, error) {
	panic(errors.New("proxyDialerWrapper.Dial should not be called directly"))
}
