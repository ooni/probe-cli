// Package tunnel allows to create tunnels to speak
// with OONI backends and other services.
package tunnel

import (
	"context"
	"errors"
	"net/url"
	"time"
)

// Session is the way in which this package sees a Session.
type Session interface {
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
	TempDir() string
}

// Tunnel is a tunnel used by the session
type Tunnel interface {
	BootstrapTime() time.Duration
	SOCKS5ProxyURL() *url.URL
	Stop()
}

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
		return nil, errors.New("unsupported tunnel")
	}
}

func enforceNilContract(tun Tunnel, err error) (Tunnel, error) {
	if err != nil {
		return nil, err
	}
	return tun, nil
}
