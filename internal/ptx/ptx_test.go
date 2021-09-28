package ptx

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestListenerLoggerWorks(t *testing.T) {
	lst := &Listener{Logger: log.Log}
	if lst.logger() != log.Log {
		t.Fatal("logger() returned an unexpected value")
	}
}

func TestListenerWorksWithFakeDialer(t *testing.T) {
	// start the fake PT
	fd := &FakeDialer{Address: "google.com:80"}
	lst := &Listener{PTDialer: fd}
	if err := lst.Start(); err != nil {
		t.Fatal(err)
	}

	// calling lst.Start again should be idempotent and race-free
	if err := lst.Start(); err != nil {
		t.Fatal(err)
	}

	// let us now _use_ the PT with a custom HTTP client.
	addr := lst.Addr()
	if addr == nil {
		t.Fatal("expected non-nil addr here")
	}
	URL := &url.URL{Scheme: "socks5", Host: addr.String()}
	clnt := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// no redirection because we force connecting to google.com:80
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{Proxy: func(r *http.Request) (*url.URL, error) {
			// force always using this proxy
			return URL, nil
		}},
	}
	resp, err := clnt.Get("http://google.com/humans.txt")
	if err != nil {
		t.Fatal(err)
	}
	data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(data))
	resp.Body.Close()
	clnt.CloseIdleConnections()

	// Stop the listener
	lst.Stop()
	lst.Stop() // should be idempotent and race free
}

func TestListenerCannotListen(t *testing.T) {
	expected := errors.New("mocked error")
	lst := &Listener{
		overrideListenSocks: func(network, laddr string) (ptxSocksListener, error) {
			return nil, expected
		},
	}
	if err := lst.Start(); !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestListenerCastListenerWorksFineOnError(t *testing.T) {
	expected := errors.New("mocked error")
	lst := &Listener{}
	out, err := lst.castListener(nil, expected)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected to see nil here")
	}
}

// mockableSocksConn is a mockable ptxSocksConn.
type mockableSocksConn struct {
	// mocks.Conn allows to mock all net.Conn functionality.
	*mocks.Conn

	// MockGrant allows to mock the Grant function.
	MockGrant func(addr *net.TCPAddr) error
}

// Grant grants access to a specific IP address.
func (c *mockableSocksConn) Grant(addr *net.TCPAddr) error {
	return c.MockGrant(addr)
}

func TestListenerHandleSocksConnWithGrantFailure(t *testing.T) {
	expected := errors.New("mocked error")
	lst := &Listener{}
	c := &mockableSocksConn{
		MockGrant: func(addr *net.TCPAddr) error {
			return expected
		},
	}
	err := lst.handleSocksConn(context.Background(), c)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

// mockableDialer is a mockable PTDialer
type mockableDialer struct {
	// MockDialContext allows to mock DialContext.
	MockDialContext func(ctx context.Context) (net.Conn, error)

	// MockAsBridgeArgument allows to mock AsBridgeArgument.
	MockAsBridgeArgument func() string

	// MockName allows to mock Name.
	MockName func() string
}

// DialContext implements PTDialer.DialContext.
func (d *mockableDialer) DialContext(ctx context.Context) (net.Conn, error) {
	return d.MockDialContext(ctx)
}

// AsBridgeArgument implements PTDialer.AsBridgeArgument.
func (d *mockableDialer) AsBridgeArgument() string {
	return d.MockAsBridgeArgument()
}

// Name implements PTDialer.Name.
func (d *mockableDialer) Name() string {
	return d.MockName()
}

var _ PTDialer = &mockableDialer{}

func TestListenerHandleSocksConnWithDialContextFailure(t *testing.T) {
	expected := errors.New("mocked error")
	d := &mockableDialer{
		MockDialContext: func(ctx context.Context) (net.Conn, error) {
			return nil, expected
		},
	}
	lst := &Listener{PTDialer: d}
	c := &mockableSocksConn{
		Conn: &mocks.Conn{
			MockClose: func() error {
				return nil
			},
		},
		MockGrant: func(addr *net.TCPAddr) error {
			return nil
		},
	}
	err := lst.handleSocksConn(context.Background(), c)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestListenerForwardWithContextWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lst := &Listener{}
	left, right := net.Pipe()
	go lst.forwardWithContext(ctx, left, right)
	cancel()
}

func TestListenerForwardWithNaturalTermination(t *testing.T) {
	lst := &Listener{}
	left, right := net.Pipe()
	go lst.forwardWithContext(context.Background(), left, right)
	right.Close()
}

// mockableSocksListener is a mockable ptxSocksListener.
type mockableSocksListener struct {
	// MockAcceptSocks allows to mock AcceptSocks.
	MockAcceptSocks func() (ptxSocksConn, error)

	// MockAddr allows to mock Addr.
	MockAddr func() net.Addr

	// MockClose allows to mock Close.
	MockClose func() error
}

// AcceptSocks implemements ptxSocksListener.AcceptSocks.
func (m *mockableSocksListener) AcceptSocks() (ptxSocksConn, error) {
	return m.MockAcceptSocks()
}

// Addr implemements ptxSocksListener.Addr.
func (m *mockableSocksListener) Addr() net.Addr {
	return m.MockAddr()
}

// Close implemements ptxSocksListener.Close.
func (m *mockableSocksListener) Close() error {
	return m.MockClose()
}

func TestListenerLoopWithTemporaryError(t *testing.T) {
	isclosed := &atomicx.Int64{}
	sl := &mockableSocksListener{
		MockAcceptSocks: func() (ptxSocksConn, error) {
			if isclosed.Load() > 0 {
				return nil, io.EOF
			}
			// this error should be temporary
			return nil, &net.OpError{
				Op:  "accept",
				Err: syscall.ECONNABORTED,
			}
		},
		MockClose: func() error {
			isclosed.Add(1)
			return nil
		},
	}
	lst := &Listener{
		cancel:   func() {},
		listener: sl,
	}
	go lst.acceptLoop(context.Background(), sl)
	time.Sleep(1 * time.Millisecond)
	lst.Stop()
}
