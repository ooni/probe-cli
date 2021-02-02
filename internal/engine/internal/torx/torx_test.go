package torx_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/torx"
)

type Closer struct {
	counter int
}

func (c *Closer) Close() error {
	c.counter++
	return errors.New("mocked mocked mocked")
}

func TestTunnelNonNil(t *testing.T) {
	closer := new(Closer)
	proxy := &url.URL{Scheme: "x", Host: "10.0.0.1:443"}
	tun := torx.NewTunnel(128, closer, proxy)
	if tun.BootstrapTime() != 128 {
		t.Fatal("not the bootstrap time we expected")
	}
	if tun.SOCKS5ProxyURL() != proxy {
		t.Fatal("not the url we expected")
	}
	tun.Stop()
	if closer.counter != 1 {
		t.Fatal("something went wrong while stopping the tunnel")
	}
}

func TestTunnelNil(t *testing.T) {
	var tun *torx.Tunnel
	if tun.BootstrapTime() != 0 {
		t.Fatal("not the bootstrap time we expected")
	}
	if tun.SOCKS5ProxyURL() != nil {
		t.Fatal("not the url we expected")
	}
	tun.Stop() // ensure we don't crash
}

func TestStartWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tun, err := torx.Start(ctx, &mockable.Session{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return nil, expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigEnableNetworkFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigGetInfoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return nil, expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigGetInfoInvalidNumberOfKeys(t *testing.T) {
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return nil, nil
		},
	})
	if err.Error() != "unable to get socks proxy address" {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigGetInfoInvalidKey(t *testing.T) {
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return []*control.KeyVal{{}}, nil
		},
	})
	if err.Error() != "unable to get socks proxy address" {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartWithConfigGetInfoInvalidProxyType(t *testing.T) {
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return []*control.KeyVal{{Key: "net/listeners/socks", Val: "127.0.0.1:9050"}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tun == nil {
		t.Fatal("expected non-nil tunnel here")
	}
}

func TestStartWithConfigSuccess(t *testing.T) {
	ctx := context.Background()
	tun, err := torx.StartWithConfig(ctx, torx.StartConfig{
		Sess: &mockable.Session{},
		Start: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		EnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		GetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return []*control.KeyVal{{Key: "net/listeners/socks", Val: "unix:/foo/bar"}}, nil
		},
	})
	if err.Error() != "tor returned unsupported proxy" {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}
