package netplumbing

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

// proxy checks whether we need to use a proxy.
func (txp *Transport) proxy(req *http.Request) (*url.URL, error) {
	ctx := req.Context()
	if settings := ContextSettings(ctx); settings != nil && settings.Proxy != nil {
		log := txp.logger(ctx)
		log.Debugf("http: using proxy: %s", settings.Proxy)
		return settings.Proxy, nil
	}
	return nil, nil
}

// ErrProxyNotImplemented indicates that we don't support connecting via proxy.
var ErrProxyNotImplemented = errors.New("netplumbing: proxy not implemented")

// proxyAdapter uses txp.connect as a child dial function
type proxyAdapter struct {
	txp *Transport
}

// DialContext implements proxy.ContextDialer.DialContext.
func (pc *proxyAdapter) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return pc.txp.connect(ctx, network, address)
}

// Dial implements proxy.Dialer.Dial.
func (pc *proxyAdapter) Dial(network, address string) (net.Conn, error) {
	panic("netplumbing: this function should not be called")
}

// proxyDialContext is a dial context that uses a proxy.
func (txp *Transport) proxyDialContext(
	ctx context.Context, proxyURL *url.URL,
	network string, address string) (net.Conn, error) {
	if proxyURL.Scheme != "socks5" {
		return nil, ErrProxyNotImplemented
	}
	var auth *proxy.Auth
	if user := proxyURL.User; user != nil {
		password, _ := user.Password()
		auth = &proxy.Auth{
			User:     user.Username(),
			Password: password,
		}
	}
	// the code at proxy/socks5.go never fails; see https://git.io/JfJ4g
	socks5, _ := proxy.SOCKS5(network, proxyURL.Host, auth, &proxyAdapter{txp})
	contextDialer := socks5.(proxy.ContextDialer)
	return contextDialer.DialContext(ctx, network, address)
}
