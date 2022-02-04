package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
)

// torCloser is used to mock a running tor process, which
// we abstract as a io.Closer in tor.go.
type torCloser struct {
	counter int
}

// Close implements io.Closer.Close.
func (c *torCloser) Close() error {
	c.counter++
	return errors.New("mocked mocked mocked")
}

func TestTorTunnelNonNil(t *testing.T) {
	closer := new(torCloser)
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

func TestTorWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorWithEmptyTunnelDir(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "",
	})
	if !errors.Is(err, ErrEmptyTunnelDir) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorBinaryNotFoundFailure(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TorBinary: "/nonexistent/directory/tor",
		TunnelDir: "testdata",
	})
	if !errors.Is(err, syscall.ENOENT) {
		t.Fatal("not the error we expected", err)
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
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

func TestTorGetProtocolInfoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
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

func TestTorEnableNetworkFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
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

func TestTorGetInfoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
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

func TestTorGetInfoInvalidNumberOfKeys(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return nil, nil
		},
	})
	if !errors.Is(err, ErrTorUnableToGetSOCKSProxyAddress) {
		t.Fatal("not the error we expected", err)
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorGetInfoInvalidKey(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return []*control.KeyVal{{}}, nil
		},
	})
	if !errors.Is(err, ErrTorUnableToGetSOCKSProxyAddress) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorGetInfoInvalidProxyType(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
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

func TestTorUnsupportedProxy(t *testing.T) {
	ctx := context.Background()
	tun, _, err := torStart(ctx, &Config{
		Session:   &MockableSession{},
		TunnelDir: "testdata",
		testExecabsLookPath: func(name string) (string, error) {
			return "/usr/local/bin/tor", nil
		},
		testTorStart: func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
			return &tor.Tor{}, nil
		},
		testTorProtocolInfo: func(tor *tor.Tor) (*control.ProtocolInfo, error) {
			return &control.ProtocolInfo{}, nil
		},
		testTorEnableNetwork: func(ctx context.Context, tor *tor.Tor, wait bool) error {
			return nil
		},
		testTorGetInfo: func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
			return []*control.KeyVal{{Key: "net/listeners/socks", Val: "unix:/foo/bar"}}, nil
		},
	})
	if !errors.Is(err, ErrTorReturnedUnsupportedProxy) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestMaybeCleanupTunnelDir(t *testing.T) {
	fakeTunDir := filepath.Join("testdata", "fake-tun-dir")
	if err := os.RemoveAll(fakeTunDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fakeTunDir, 0700); err != nil {
		t.Fatal(err)
	}
	fakeData := []byte("deadbeef\n")
	logfile := filepath.Join(fakeTunDir, "tor.log")
	if err := os.WriteFile(logfile, fakeData, 0600); err != nil {
		t.Fatal(err)
	}
	for idx := 0; idx < 3; idx++ {
		filename := filepath.Join(fakeTunDir, fmt.Sprintf("torrc-%d", idx))
		if err := os.WriteFile(filename, fakeData, 0600); err != nil {
			t.Fatal(err)
		}
		filename = filepath.Join(fakeTunDir, fmt.Sprintf("control-port-%d", idx))
		if err := os.WriteFile(filename, fakeData, 0600); err != nil {
			t.Fatal(err)
		}
		filename = filepath.Join(fakeTunDir, fmt.Sprintf("antani-%d", idx))
		if err := os.WriteFile(filename, fakeData, 0600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := filepath.Glob(filepath.Join(fakeTunDir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 10 {
		t.Fatal("unexpected number of files")
	}
	maybeCleanupTunnelDir(fakeTunDir, logfile)
	files, err = filepath.Glob(filepath.Join(fakeTunDir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatal("unexpected number of files")
	}
	expectPrefix := filepath.Join(fakeTunDir, "antani-")
	for _, file := range files {
		if !strings.HasPrefix(file, expectPrefix) {
			t.Fatal("unexpected file name: ", file)
		}
	}
}
