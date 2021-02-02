package dialer

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

type ProxyDialerWrapper = proxyDialerWrapper

func (d ProxyDialer) DialContextWithDialer(
	ctx context.Context, child proxy.Dialer, network, address string) (net.Conn, error) {
	return d.dial(ctx, child, network, address)
}
