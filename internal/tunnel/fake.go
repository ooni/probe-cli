package tunnel

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/armon/go-socks5"
)

// fakeTunnel is a fake tunnel.
type fakeTunnel struct {
	addr          net.Addr
	bootstrapTime time.Duration
	listener      net.Listener
	once          sync.Once
}

// BootstrapTime implements Tunnel.BootstrapTime.
func (t *fakeTunnel) BootstrapTime() time.Duration {
	return t.bootstrapTime
}

// Stop implements Tunnel.Stop.
func (t *fakeTunnel) Stop() {
	// Implementation note: closing the listener causes
	// the socks5 server.Serve to return an error
	t.once.Do(func() { _ = t.listener.Close() })
}

// SOCKS5ProxyURL returns the SOCKS5 proxy URL.
func (t *fakeTunnel) SOCKS5ProxyURL() *url.URL {
	return &url.URL{
		Scheme: "socks5",
		Host:   t.addr.String(),
	}
}

// fakeStart starts the fake tunnel.
func fakeStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	// do the same things other tunnels do:
	//
	// 1. abort if context is cancelled
	//
	// 2. check for tunnelDir being not empty
	//
	// 3. attempt to create tunnelDir
	//
	// after that, it's all fake and we just create a simple
	// socks5 server that we can use
	debugInfo := DebugInfo{
		LogFilePath: "",
		Name:        "fake",
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
	if err := config.mkdirAll(config.TunnelDir, 0700); err != nil {
		return nil, debugInfo, err
	}
	server, err := config.socks5New(&socks5.Config{})
	if err != nil {
		return nil, debugInfo, err
	}
	start := time.Now()
	listener, err := config.netListen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, debugInfo, err
	}
	bootstrapTime := time.Since(start)
	go server.Serve(listener)
	return &fakeTunnel{
		addr:          listener.Addr(),
		bootstrapTime: bootstrapTime,
		listener:      listener,
	}, debugInfo, nil
}
