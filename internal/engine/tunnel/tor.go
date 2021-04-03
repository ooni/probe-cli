package tunnel

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
)

// torProcess is a running tor process
type torProcess interface {
	Close() error
}

// torTunnel is the Tor tunnel
type torTunnel struct {
	bootstrapTime time.Duration
	instance      torProcess
	proxy         *url.URL
}

// BootstrapTime is the bootstrsap time
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

// torStartConfig contains the configuration for StartWithConfig
type torStartConfig struct {
	Sess          Session
	Start         func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error)
	EnableNetwork func(ctx context.Context, tor *tor.Tor, wait bool) error
	GetInfo       func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error)
}

// torStart starts the tor tunnel
func torStart(ctx context.Context, sess Session) (Tunnel, error) {
	return torStartWithConfig(ctx, torStartConfig{
		Sess: sess,
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return tor.Start(ctx, conf)
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return tor.EnableNetwork(ctx, wait)
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return ctrl.GetInfo(keys...)
		},
	})
}

// torStartWithConfig is a configurable torStart for testing
func torStartWithConfig(ctx context.Context, config torStartConfig) (Tunnel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // allows to write unit tests using this code
	default:
	}
	logfile := LogFile(config.Sess)
	extraArgs := append([]string{}, config.Sess.TorArgs()...)
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, "notice stderr")
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, fmt.Sprintf(`notice file %s`, logfile))
	instance, err := config.Start(ctx, &tor.StartConf{
		DataDir:   path.Join(config.Sess.TempDir(), "tor"),
		ExtraArgs: extraArgs,
		ExePath:   config.Sess.TorBinary(),
		NoHush:    true,
	})
	if err != nil {
		return nil, err
	}
	instance.StopProcessOnClose = true
	start := time.Now()
	if err := config.EnableNetwork(ctx, instance, true); err != nil {
		instance.Close()
		return nil, err
	}
	stop := time.Now()
	// Adapted from <https://git.io/Jfc7N>
	info, err := config.GetInfo(instance.Control, "net/listeners/socks")
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

// newTorTunnel creates a new torTunnel
func newTorTunnel(bootstrapTime time.Duration, instance torProcess, proxy *url.URL) *torTunnel {
	return &torTunnel{
		bootstrapTime: bootstrapTime,
		instance:      instance,
		proxy:         proxy,
	}
}
