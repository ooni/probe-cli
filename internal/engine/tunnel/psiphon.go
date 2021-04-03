package tunnel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"time"

	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// psiphonTunnel is a psiphon tunnel
type psiphonTunnel struct {
	// tunnel is the underlying psiphon tunnel
	tunnel *clientlib.PsiphonTunnel

	// duration is the duration of the bootstrap
	duration time.Duration
}

// TODO(bassosimone): _always_ wiping the state directory
// here is absolutely wrong. This prevents us from reusing
// an existing psiphon cache existing on disk. We want to
// delete the directory _only_ in the psiphon nettest.

// psiphonMakeWorkingDir creates the working directory
func psiphonMakeWorkingDir(config *Config) (string, error) {
	const testdirname = "oonipsiphon"
	baseWorkDir := config.WorkDir
	if baseWorkDir == "" {
		baseWorkDir = config.Session.TempDir()
	}
	workdir := filepath.Join(baseWorkDir, testdirname)
	if err := config.removeAll(workdir); err != nil {
		return "", err
	}
	if err := config.mkdirAll(workdir, 0700); err != nil {
		return "", err
	}
	return workdir, nil
}

// psiphonStart starts the psiphon tunnel.
func psiphonStart(ctx context.Context, config *Config) (Tunnel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // simplifies unit testing this code
	default:
	}
	configJSON, err := config.Session.FetchPsiphonConfig(ctx)
	if err != nil {
		return nil, err
	}
	workdir, err := psiphonMakeWorkingDir(config)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	tunnel, err := config.startPsiphon(ctx, configJSON, workdir)
	if err != nil {
		return nil, err
	}
	stop := time.Now()
	return &psiphonTunnel{tunnel: tunnel, duration: stop.Sub(start)}, nil
}

// TODO(bassosimone): define the NullTunnel rather than relying on
// this magic that a nil psiphonTunnel works.

// Stop is an idempotent method that shuts down the tunnel
func (t *psiphonTunnel) Stop() {
	if t != nil {
		t.tunnel.Stop()
	}
}

// SOCKS5ProxyURL returns the SOCKS5 proxy URL.
func (t *psiphonTunnel) SOCKS5ProxyURL() (proxyURL *url.URL) {
	if t != nil {
		proxyURL = &url.URL{
			Scheme: "socks5",
			Host: net.JoinHostPort(
				"127.0.0.1", fmt.Sprintf("%d", t.tunnel.SOCKSProxyPort)),
		}
	}
	return
}

// BootstrapTime returns the bootstrap time
func (t *psiphonTunnel) BootstrapTime() (duration time.Duration) {
	if t != nil {
		duration = t.duration
	}
	return
}
