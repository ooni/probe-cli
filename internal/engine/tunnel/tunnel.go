// Package tunnel allows to create tunnels to speak
// with OONI backends and other services.
//
// You need to fill a Config object and call Start to
// obtain an instance of Tunnel. The tunnel will expose
// a SOCKS5 proxy. You need to configure your HTTP
// code to use such a proxy. Remember to call the Stop
// method of a tunnel when you are done.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Session is a measurement session. We filter for the only
// functionality we're interested to use. That is, fetching the
// psiphon configuration from the OONI backend (if possible).
type Session interface {
	// FetchPsiphonConfig should fetch and return the psiphon config
	// as a serialized JSON, or fail with an error.
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
}

// Tunnel is a tunnel for communicating with OONI backends
// (and other services) to circumvent blocking.
type Tunnel interface {
	// BootstrapTime returns the time it required to
	// create a new tunnel instance.
	BootstrapTime() time.Duration

	// SOCKS5ProxyURL returns the SOCSK5 proxy URL.
	SOCKS5ProxyURL() *url.URL

	// Stop stops the tunnel. You should not attempt to
	// use any other tunnel method after Stop.
	Stop()
}

// ErrEmptyTunnelDir indicates that config.TunnelDir is empty.
var ErrEmptyTunnelDir = errors.New("TunnelDir is empty")

// ErrUnsupportedTunnelName indicates that the given tunnel name
// is not supported by this package.
var ErrUnsupportedTunnelName = errors.New("unsupported tunnel name")

// Start starts a new tunnel by name or returns an error. We currently
// support the following tunnels:
//
// The "tor" tunnel requires the "tor" binary to be installed on
// your system. You can use config.TorArgs and config.TorBinary to
// select what binary to execute and with which arguments.
//
// The "psiphon" tunnel requires a configuration. Some builds of
// ooniprobe embed a configuration into the binary. If there is no
// embedded configuration, we will try to use config.Session to
// fetch such a configuration. This step will fail if the OONI
// backend is not reachable for any reason.
func Start(ctx context.Context, config *Config) (Tunnel, error) {
	switch config.Name {
	case "psiphon":
		return psiphonStart(ctx, config)
	case "tor":
		return torStart(ctx, config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedTunnelName, config.Name)
	}
}
