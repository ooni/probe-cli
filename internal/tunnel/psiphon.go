package tunnel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"time"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// mockableStartPsiphon allows us to test for psiphon startup failures.
var mockableStartPsiphon = func(
	ctx context.Context, config []byte, workdir string) (*clientlib.PsiphonTunnel, error) {
	return clientlib.StartTunnel(ctx, config, "", clientlib.Parameters{
		DataRootDirectory: &workdir}, nil, nil)
}

// psiphonTunnel is a psiphon tunnel
type psiphonTunnel struct {
	// bootstrapTime is the bootstrapTime of the bootstrap
	bootstrapTime time.Duration

	// tunnel is the underlying psiphon tunnel
	tunnel *clientlib.PsiphonTunnel
}

// psiphonMakeWorkingDir creates the working directory
func psiphonMakeWorkingDir(config *Config) (string, error) {
	workdir := filepath.Join(config.TunnelDir, config.Name)
	if err := config.mkdirAll(workdir, 0700); err != nil {
		return "", err
	}
	return workdir, nil
}

// psiphonStart starts the psiphon tunnel.
func psiphonStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	debugInfo := DebugInfo{
		LogFilePath: "",
		Name:        "psiphon",
		Version:     "",
	}
	select {
	case <-ctx.Done():
		return nil, debugInfo, ctx.Err() // simplifies unit testing this code
	default:
	}
	if config.TunnelDir == "" {
		return nil, debugInfo, ErrEmptyTunnelDir
	}
	configJSON, err := config.Session.FetchPsiphonConfig(ctx)
	if err != nil {
		return nil, debugInfo, err
	}
	workdir, err := psiphonMakeWorkingDir(config)
	if err != nil {
		return nil, debugInfo, err
	}
	start := time.Now()
	tunnel, err := mockableStartPsiphon(ctx, configJSON, workdir)
	if err != nil {
		return nil, debugInfo, err
	}
	stop := time.Now()
	return &psiphonTunnel{
		tunnel:        tunnel,
		bootstrapTime: stop.Sub(start),
	}, debugInfo, nil
}

// Stop is an idempotent method that shuts down the tunnel
func (t *psiphonTunnel) Stop() {
	t.tunnel.Stop()
}

// SOCKS5ProxyURL returns the SOCKS5 proxy URL.
func (t *psiphonTunnel) SOCKS5ProxyURL() *url.URL {
	return &url.URL{
		Scheme: "socks5",
		Host: net.JoinHostPort(
			"127.0.0.1", fmt.Sprintf("%d", t.tunnel.SOCKSProxyPort)),
	}
}

// BootstrapTime returns the bootstrap time
func (t *psiphonTunnel) BootstrapTime() time.Duration {
	return t.bootstrapTime
}
