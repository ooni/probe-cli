package tunnel

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
)

type Closer struct {
	counter int
}

func (c *Closer) Close() error {
	c.counter++
	return errors.New("mocked mocked mocked")
}

func TestTorTunnelNonNil(t *testing.T) {
	closer := new(Closer)
	proxy := &url.URL{Scheme: "x", Host: "10.0.0.1:443"}
	tun := &torTunnel{
		bootstrapTime: 128,
		instance:      closer,
		proxy:         proxy,
	}
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

func TestTorTunnelNil(t *testing.T) {
	var tun *torTunnel
	if tun.BootstrapTime() != 0 {
		t.Fatal("not the bootstrap time we expected")
	}
	if tun.SOCKS5ProxyURL() != nil {
		t.Fatal("not the url we expected")
	}
	tun.Stop() // ensure we don't crash
}

func TestTorStartWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tun, err := torStart(ctx, &Config{Session: &mockable.Session{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorStartStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
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

func TestTorStartEnableNetworkFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
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

func TestTorStartGetInfoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
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

func TestTorStartGetInfoInvalidNumberOfKeys(t *testing.T) {
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
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

func TestTorStartGetInfoInvalidKey(t *testing.T) {
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
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

func TestTorStartGetInfoInvalidProxyType(t *testing.T) {
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
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

func TestTorStartUnsupportedProxy(t *testing.T) {
	ctx := context.Background()
	tun, err := torStart(ctx, &Config{
		Session: &mockable.Session{},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
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
