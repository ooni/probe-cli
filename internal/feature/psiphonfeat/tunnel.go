package psiphonfeat

import "errors"

// Tunnel is the interface implementing the Psiphon tunnel.
type Tunnel interface {
	// Stop stops a running Psiphon tunnel.
	Stop()

	// GetSOCKSProxyPort returns the SOCKS5 port used by the tunnel.
	GetSOCKSProxyPort() int
}

// ErrFeatureNotEnabled indicates that the Psiphon feature is not enabled in this build.
var ErrFeatureNotEnabled = errors.New("psiphonfeat: not enabled")
