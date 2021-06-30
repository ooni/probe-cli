package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestTLSHandshakerConfigurableWithError(t *testing.T) {
	var times []time.Time
	h := &TLSHandshakerConfigurable{}
	tcpConn := &netxmocks.Conn{
		MockWrite: func(b []byte) (int, error) {
			return 0, io.EOF
		},
		MockSetDeadline: func(t time.Time) error {
			times = append(times, t)
			return nil
		},
	}
	ctx := context.Background()
	conn, _, err := h.Handshake(ctx, tcpConn, &tls.Config{
		ServerName: "x.org",
	})
	if err != io.EOF {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	if len(times) != 2 {
		t.Fatal("expected two time entries")
	}
	if !times[0].After(time.Now()) {
		t.Fatal("timeout not in the future")
	}
	if !times[1].IsZero() {
		t.Fatal("did not clear timeout on exit")
	}
}

func TestTLSHandshakerConfigurableSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(200)
	})
	srvr := httptest.NewTLSServer(handler)
	defer srvr.Close()
	URL, err := url.Parse(srvr.URL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", URL.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	handshaker := &TLSHandshakerConfigurable{}
	ctx := context.Background()
	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		ServerName:         URL.Hostname(),
	}
	tlsConn, connState, err := handshaker.Handshake(ctx, conn, config)
	if err != nil {
		t.Fatal(err)
	}
	defer tlsConn.Close()
	if connState.Version != tls.VersionTLS13 {
		t.Fatal("unexpected TLS version")
	}
}

func TestTLSHandshakerConfigurableSetsDefaultRootCAs(t *testing.T) {
	expected := errors.New("mocked error")
	var gotTLSConfig *tls.Config
	handshaker := &TLSHandshakerConfigurable{
		NewConn: func(conn net.Conn, config *tls.Config) TLSConn {
			gotTLSConfig = config
			return &netxmocks.TLSConn{
				MockHandshake: func() error {
					return expected
				},
			}
		},
	}
	ctx := context.Background()
	config := &tls.Config{}
	conn := &netxmocks.Conn{
		MockSetDeadline: func(t time.Time) error {
			return nil
		},
	}
	tlsConn, connState, err := handshaker.Handshake(ctx, conn, config)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if !reflect.ValueOf(connState).IsZero() {
		t.Fatal("expected zero connState here")
	}
	if tlsConn != nil {
		t.Fatal("expected nil tlsConn here")
	}
	if config.RootCAs != nil {
		t.Fatal("config.RootCAs should still be nil")
	}
	if gotTLSConfig.RootCAs != defaultCertPool {
		t.Fatal("gotTLSConfig.RootCAs has not been correctly set")
	}
}

func TestTLSHandshakerLoggerSuccess(t *testing.T) {
	th := &TLSHandshakerLogger{
		TLSHandshaker: &netxmocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return tls.Client(conn, config), tls.ConnectionState{}, nil
			},
		},
		Logger: log.Log,
	}
	conn := &netxmocks.Conn{
		MockClose: func() error {
			return nil
		},
	}
	config := &tls.Config{}
	ctx := context.Background()
	tlsConn, connState, err := th.Handshake(ctx, conn, config)
	if err != nil {
		t.Fatal(err)
	}
	if err := tlsConn.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.ValueOf(connState).IsZero() {
		t.Fatal("expected zero ConnectionState here")
	}
}

func TestTLSHandshakerLoggerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	th := &TLSHandshakerLogger{
		TLSHandshaker: &netxmocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return nil, tls.ConnectionState{}, expected
			},
		},
		Logger: log.Log,
	}
	conn := &netxmocks.Conn{
		MockClose: func() error {
			return nil
		},
	}
	config := &tls.Config{}
	ctx := context.Background()
	tlsConn, connState, err := th.Handshake(ctx, conn, config)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if tlsConn != nil {
		t.Fatal("expected nil conn here")
	}
	if !reflect.ValueOf(connState).IsZero() {
		t.Fatal("expected zero ConnectionState here")
	}
}
