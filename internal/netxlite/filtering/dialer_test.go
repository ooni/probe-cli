package filtering

import "github.com/ooni/probe-cli/v3/internal/netxlite"

// tProxyDialerAdapter adapts a netxlite.TProxyDialer to be a netxlite.Dialer.
type tProxyDialerAdapter struct {
	netxlite.TProxyDialer
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (*tProxyDialerAdapter) CloseIdleConnections() {
	// nothing
}
