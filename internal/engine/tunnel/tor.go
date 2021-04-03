package tunnel

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cretz/bine/tor"
)

// torProcess is a running tor process
type torProcess interface {
	// Close kills the running tor process
	Close() error
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
func (tt *torTunnel) BootstrapTime() (duration time.Duration) {
	if tt != nil {
		duration = tt.bootstrapTime
	}
	return
}

// SOCKS5ProxyURL returns the URL of the SOCKS5 proxy
func (tt *torTunnel) SOCKS5ProxyURL() (url *url.URL) {
	if tt != nil {
		url = tt.proxy
	}
	return
}

// Stop stops the Tor tunnel
func (tt *torTunnel) Stop() {
	if tt != nil {
		tt.instance.Close()
	}
}

// torStart starts the tor tunnel.
func torStart(ctx context.Context, config *Config) (Tunnel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // allows to write unit tests using this code
	default:
	}
	logfile := LogFile(config.Session)
	extraArgs := append([]string{}, config.Session.TorArgs()...)
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, "notice stderr")
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, fmt.Sprintf(`notice file %s`, logfile))
	instance, err := config.torStart(ctx, &tor.StartConf{
		DataDir:   path.Join(config.Session.TempDir(), "tor"),
		ExtraArgs: extraArgs,
		ExePath:   config.Session.TorBinary(),
		NoHush:    true,
	})
	if err != nil {
		return nil, err
	}
	instance.StopProcessOnClose = true
	start := time.Now()
	if err := config.torEnableNetwork(ctx, instance, true); err != nil {
		instance.Close()
		return nil, err
	}
	stop := time.Now()
	// Adapted from <https://git.io/Jfc7N>
	info, err := config.torGetInfo(instance.Control, "net/listeners/socks")
	if err != nil {
		instance.Close()
		return nil, err
	}
	if len(info) != 1 || info[0].Key != "net/listeners/socks" {
		instance.Close()
		return nil, fmt.Errorf("unable to get socks proxy address")
	}
	proxyAddress := info[0].Val
	if strings.HasPrefix(proxyAddress, "unix:") {
		instance.Close()
		return nil, fmt.Errorf("tor returned unsupported proxy")
	}
	return &torTunnel{
		bootstrapTime: stop.Sub(start),
		instance:      instance,
		proxy:         &url.URL{Scheme: "socks5", Host: proxyAddress},
	}, nil
}

// LogFile returns the name of tor logs given a specific session. The file
// is always located somewhere inside the sess.TempDir() directory.
func LogFile(sess Session) string {
	return path.Join(sess.TempDir(), "tor.log")
}
