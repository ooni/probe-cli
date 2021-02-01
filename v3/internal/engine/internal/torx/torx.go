// Package torx contains code to control tor.
package torx

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

// Session is the way in which this package sees a Session.
type Session interface {
	TempDir() string
	TorArgs() []string
	TorBinary() string
}

// TorProcess is a running tor process
type TorProcess interface {
	Close() error
}

// Tunnel is the Tor tunnel
type Tunnel struct {
	bootstrapTime time.Duration
	instance      TorProcess
	proxy         *url.URL
}

// BootstrapTime is the bootstrsap time
func (tt *Tunnel) BootstrapTime() (duration time.Duration) {
	if tt != nil {
		duration = tt.bootstrapTime
	}
	return
}

// SOCKS5ProxyURL returns the URL of the SOCKS5 proxy
func (tt *Tunnel) SOCKS5ProxyURL() (url *url.URL) {
	if tt != nil {
		url = tt.proxy
	}
	return
}

// Stop stops the Tor tunnel
func (tt *Tunnel) Stop() {
	if tt != nil {
		tt.instance.Close()
	}
}

// StartConfig contains the configuration for StartWithConfig
type StartConfig struct {
	Sess          Session
	Start         func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error)
	EnableNetwork func(ctx context.Context, tor *tor.Tor, wait bool) error
	GetInfo       func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error)
}

// Start starts the tor tunnel
func Start(ctx context.Context, sess Session) (*Tunnel, error) {
	return StartWithConfig(ctx, StartConfig{
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

// StartWithConfig is a configurable Start for testing
func StartWithConfig(ctx context.Context, config StartConfig) (*Tunnel, error) {
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
	return &Tunnel{
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
