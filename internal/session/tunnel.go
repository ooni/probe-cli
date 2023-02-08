package session

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// ErrUnsupportedTunnelScheme indicates we don't support the tunnel scheme.
var ErrUnsupportedTunnelScheme = errors.New("session: unsupported tunnel scheme")

// newTunnel creates a new [tunnel.Tunnel] given a ProxyURL.
func newTunnel(ctx context.Context, logger model.Logger, req *BootstrapRequest) (tunnel.Tunnel, error) {
	if req.ProxyURL == "" {
		logger.Info("no need to create any tunnel")
		return &nullTunnel{}, nil
	}
	URL, err := url.Parse(req.ProxyURL)
	if err != nil {
		return nil, err
	}
	switch scheme := URL.Scheme; scheme {
	case "socks5":
		logger.Infof("creating fake tunnel for %s", req.ProxyURL)
		return &socks5Tunnel{URL}, nil
	case "tor", "torsf":
		logger.Infof("creating %s tunnel; please, be patient...", scheme)
		return newTorOrTorsfTunnel(ctx, logger, req, scheme)
	case "psiphon":
		logger.Info("creating psiphon tunnel; please, be patient...")
		return newPsiphonTunnel(ctx, logger, req)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedTunnelScheme, scheme)
	}
}

// nullTunnel is the absence of any tunnel.
type nullTunnel struct{}

// BootstrapTime implements tunnel.Tunnel
func (t *nullTunnel) BootstrapTime() time.Duration {
	return 0
}

// SOCKS5ProxyURL implements tunnel.Tunnel
func (t *nullTunnel) SOCKS5ProxyURL() *url.URL {
	return nil // perfectly fine to return nil in this context
}

// Stop implements tunnel.Tunnel
func (t *nullTunnel) Stop() {
	// nothing
}

// socks5Tunnel is a fake tunnel that returns a SOCKS5 URL.
type socks5Tunnel struct {
	url *url.URL
}

// BootstrapTime implements tunnel.Tunnel
func (t *socks5Tunnel) BootstrapTime() time.Duration {
	return 0
}

// SOCKS5ProxyURL implements tunnel.Tunnel
func (t *socks5Tunnel) SOCKS5ProxyURL() *url.URL {
	return t.url
}

// Stop implements tunnel.Tunnel
func (t *socks5Tunnel) Stop() {
	// nothing
}

// newTorOrTorsfTunnel creates a tor or torsf tunnel.
func newTorOrTorsfTunnel(
	ctx context.Context,
	logger model.Logger,
	req *BootstrapRequest,
	scheme string,
) (tunnel.Tunnel, error) {
	config := &tunnel.Config{
		Name:                scheme,
		Session:             &engine.SessionTunnelEarlySession{},
		SnowflakeRendezvous: req.SnowflakeRendezvousMethod,
		TunnelDir:           req.TunnelDir,
		Logger:              logger,
		TorArgs:             req.TorArgs,
		TorBinary:           req.TorBinary,
	}
	tun, _, err := tunnel.Start(ctx, config)
	return tun, err
}

// newPsiphonTunnel creates a psiphon tunnel.
func newPsiphonTunnel(
	ctx context.Context,
	logger model.Logger,
	req *BootstrapRequest,
) (tunnel.Tunnel, error) {
	config := &tunnel.Config{
		Name:                "psiphon",
		Session:             &engine.SessionTunnelEarlySession{},
		SnowflakeRendezvous: "", // not needed
		TunnelDir:           req.TunnelDir,
		Logger:              logger,
		TorArgs:             nil, // not needed
		TorBinary:           "",  // not needed
	}
	tun, _, err := tunnel.Start(ctx, config)
	return tun, err
}
