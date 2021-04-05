// Package tunnel allows to create tunnels to speak
// with OONI backends and other services.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Session is the way in which this package sees a Session.
type Session interface {
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
}

// Tunnel is a tunnel used by the session
type Tunnel interface {
	// BootstrapTime returns the time it required to
	// create an instance of the tunnel
	BootstrapTime() time.Duration

	// SOCKS5ProxyURL returns the SOCSK5 proxy URL
	SOCKS5ProxyURL() *url.URL

	// Stop stops the tunnel. This method is idempotent.
	Stop()
}

// ErrEmptyTunnelDir indicates that config.TunnelDir is empty.
var ErrEmptyTunnelDir = errors.New("TunnelDir is empty")

// ErrUnsupportedTunnelName indicates that the given tunnel name
// is not supported by this package.
var ErrUnsupportedTunnelName = errors.New("unsupported tunnel name")

// Start starts a new tunnel by name or returns an error. Note that if you
// pass to this function the "" tunnel, you get back nil, nil.
func Start(ctx context.Context, config *Config) (Tunnel, error) {
	switch config.Name {
	case "":
		return enforceNilContract(nil, nil)
	case "psiphon":
		tun, err := psiphonStart(ctx, config)
		return enforceNilContract(tun, err)
	case "tor":
		tun, err := torStart(ctx, config)
		return enforceNilContract(tun, err)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedTunnelName, config.Name)
	}
}

// enforceNilContract ensures that either the tunnel is nil
// or the error is nil.
func enforceNilContract(tun Tunnel, err error) (Tunnel, error) {
	// TODO(bassosimone): we currently allow returning nil, nil but
	// we want to change this to return a fake NilTunnel.
	if err != nil {
		return nil, err
	}
	return tun, nil
}
