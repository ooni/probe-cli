package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrCannotFindTorBinary is an error emitted when we cannot find the
// tor binary. We use this error in vanillator and torsf to detect
// cases where this happens and avoid submitting measurements.
var ErrCannotFindTorBinary = errors.New("tunnel: cannot find tor binary")

// torProcess is a running tor process.
type torProcess interface {
	io.Closer
}

// torTunnel is the Tor tunnel
type torTunnel struct {
	// bootstrapTime is the duration of the bootstrap
	bootstrapTime time.Duration

	// instance is the running tor instance
	instance torProcess

	// proxy is the SOCKS5 proxy URL
	proxy *url.URL
}

// BootstrapTime returns the bootstrap time
func (tt *torTunnel) BootstrapTime() time.Duration {
	return tt.bootstrapTime
}

// SOCKS5ProxyURL returns the URL of the SOCKS5 proxy
func (tt *torTunnel) SOCKS5ProxyURL() *url.URL {
	return tt.proxy
}

// Stop stops the Tor tunnel
func (tt *torTunnel) Stop() {
	_ = tt.instance.Close()
}

// ErrTorUnableToGetSOCKSProxyAddress indicates that we could not
// get the SOCKS proxy address via the control port.
var ErrTorUnableToGetSOCKSProxyAddress = errors.New(
	"unable to get socks proxy address")

// ErrTorReturnedUnsupportedProxy indicates that tor returned to
// us the address of a proxy that we don't support.
var ErrTorReturnedUnsupportedProxy = errors.New(
	"tor returned unsupported proxy")

// torStart starts the tor tunnel.
func torStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	debugInfo := DebugInfo{
		LogFilePath: "",
		Name:        "tor",
		Version:     "",
	}
	select {
	case <-ctx.Done():
		return nil, debugInfo, ctx.Err() // allows to write unit tests using this code
	default:
	}
	if config.TunnelDir == "" {
		return nil, debugInfo, ErrEmptyTunnelDir
	}
	stateDir := filepath.Join(config.TunnelDir, "tor")
	logfile := filepath.Join(stateDir, "tor.log")
	debugInfo.LogFilePath = logfile
	maybeCleanupTunnelDir(stateDir, logfile)
	extraArgs := append([]string{}, config.TorArgs...)
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, "notice stderr")
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, fmt.Sprintf(`notice file %s`, logfile))
	torStartConf, err := getTorStartConf(config, stateDir, extraArgs)
	if err != nil {
		return nil, debugInfo, err
	}
	instance, err := config.torStart(ctx, torStartConf)
	if err != nil {
		return nil, debugInfo, err
	}
	protoInfo, err := config.torProtocolInfo(instance)
	if err != nil {
		return nil, debugInfo, err
	}
	debugInfo.Version = protoInfo.TorVersion
	instance.StopProcessOnClose = true
	start := time.Now()
	if err := config.torEnableNetwork(ctx, instance, true); err != nil {
		_ = instance.Close()
		return nil, debugInfo, err
	}
	stop := time.Now()
	// Adapted from <https://git.io/Jfc7N>
	info, err := config.torGetInfo(instance.Control, "net/listeners/socks")
	if err != nil {
		_ = instance.Close()
		return nil, debugInfo, err
	}
	if len(info) != 1 || info[0].Key != "net/listeners/socks" {
		_ = instance.Close()
		return nil, debugInfo, ErrTorUnableToGetSOCKSProxyAddress
	}
	proxyAddress := info[0].Val
	if strings.HasPrefix(proxyAddress, "unix:") {
		_ = instance.Close()
		return nil, debugInfo, ErrTorReturnedUnsupportedProxy
	}
	return &torTunnel{
		bootstrapTime: stop.Sub(start),
		instance:      instance,
		proxy:         &url.URL{Scheme: "socks5", Host: proxyAddress},
	}, debugInfo, nil
}

// maybeCleanupTunnelDir removes stale files inside
// of the tunnel directory.
func maybeCleanupTunnelDir(dir, logfile string) {
	_ = os.Remove(logfile)
	removeWithGlob(filepath.Join(dir, "torrc-*"))
	removeWithGlob(filepath.Join(dir, "control-port-*"))
}

// removeWithGlob globs and removes files.
func removeWithGlob(pattern string) {
	files, _ := filepath.Glob(pattern)
	for _, file := range files {
		_ = os.Remove(file)
	}
}
