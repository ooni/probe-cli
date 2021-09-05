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
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestVersionString(t *testing.T) {
	if TLSVersionString(tls.VersionTLS13) != "TLSv1.3" {
		t.Fatal("not working for existing version")
	}
	if TLSVersionString(1) != "TLS_VERSION_UNKNOWN_1" {
		t.Fatal("not working for nonexisting version")
	}
	if TLSVersionString(0) != "" {
		t.Fatal("not working for zero version")
	}
}

func TestCipherSuite(t *testing.T) {
	if TLSCipherSuiteString(tls.TLS_AES_128_GCM_SHA256) != "TLS_AES_128_GCM_SHA256" {
		t.Fatal("not working for existing cipher suite")
	}
	if TLSCipherSuiteString(1) != "TLS_CIPHER_SUITE_UNKNOWN_1" {
		t.Fatal("not working for nonexisting cipher suite")
	}
	if TLSCipherSuiteString(0) != "" {
		t.Fatal("not working for zero cipher suite")
	}
}

func TestNewDefaultCertPoolWorks(t *testing.T) {
	pool := NewDefaultCertPool()
	if pool == nil {
		t.Fatal("expected non-nil value here")
	}
}

func TestConfigureTLSVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		wantErr    error
		versionMin int
		versionMax int
	}{{
		name:       "with TLSv1.3",
		version:    "TLSv1.3",
		wantErr:    nil,
		versionMin: tls.VersionTLS13,
		versionMax: tls.VersionTLS13,
	}, {
		name:       "with TLSv1.2",
		version:    "TLSv1.2",
		wantErr:    nil,
		versionMin: tls.VersionTLS12,
		versionMax: tls.VersionTLS12,
	}, {
		name:       "with TLSv1.1",
		version:    "TLSv1.1",
		wantErr:    nil,
		versionMin: tls.VersionTLS11,
		versionMax: tls.VersionTLS11,
	}, {
		name:       "with TLSv1.0",
		version:    "TLSv1.0",
		wantErr:    nil,
		versionMin: tls.VersionTLS10,
		versionMax: tls.VersionTLS10,
	}, {
		name:       "with TLSv1",
		version:    "TLSv1",
		wantErr:    nil,
		versionMin: tls.VersionTLS10,
		versionMax: tls.VersionTLS10,
	}, {
		name:       "with default",
		version:    "",
		wantErr:    nil,
		versionMin: 0,
		versionMax: 0,
	}, {
		name:       "with invalid version",
		version:    "TLSv999",
		wantErr:    ErrInvalidTLSVersion,
		versionMin: 0,
		versionMax: 0,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := new(tls.Config)
			err := ConfigureTLSVersion(conf, tt.version)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("not the error we expected: %+v", err)
			}
			if conf.MinVersion != uint16(tt.versionMin) {
				t.Fatalf("not the min version we expected: %+v", conf.MinVersion)
			}
			if conf.MaxVersion != uint16(tt.versionMax) {
				t.Fatalf("not the max version we expected: %+v", conf.MaxVersion)
			}
		})
	}
}

func TestTLSHandshakerConfigurableWithError(t *testing.T) {
	var times []time.Time
	h := &TLSHandshakerConfigurable{}
	tcpConn := &mocks.Conn{
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
			return &mocks.TLSConn{
				MockHandshake: func() error {
					return expected
				},
			}
		},
	}
	ctx := context.Background()
	config := &tls.Config{}
	conn := &mocks.Conn{
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
		TLSHandshaker: &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return tls.Client(conn, config), tls.ConnectionState{}, nil
			},
		},
		Logger: log.Log,
	}
	conn := &mocks.Conn{
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
		TLSHandshaker: &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return nil, tls.ConnectionState{}, expected
			},
		},
		Logger: log.Log,
	}
	conn := &mocks.Conn{
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

func TestTLSDialerFailureSplitHostPort(t *testing.T) {
	dialer := &TLSDialer{}
	ctx := context.Background()
	const address = "www.google.com" // missing port
	conn, err := dialer.DialTLSContext(ctx, "tcp", address)
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureDialing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	dialer := TLSDialer{Dialer: &net.Dialer{}}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureHandshaking(t *testing.T) {
	ctx := context.Background()
	dialer := TLSDialer{
		Config: &tls.Config{},
		Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return &mocks.Conn{MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			}, MockClose: func() error {
				return nil
			}, MockSetDeadline: func(t time.Time) error {
				return nil
			}}, nil
		}},
		TLSHandshaker: &TLSHandshakerConfigurable{},
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerSuccessHandshaking(t *testing.T) {
	ctx := context.Background()
	dialer := TLSDialer{
		Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return &mocks.Conn{MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			}, MockClose: func() error {
				return nil
			}, MockSetDeadline: func(t time.Time) error {
				return nil
			}}, nil
		}},
		TLSHandshaker: &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return tls.Client(conn, config), tls.ConnectionState{}, nil
			},
		},
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("connection is nil")
	}
	conn.Close()
}

func TestTLSDialerConfigFromEmptyConfigForWeb(t *testing.T) {
	d := &TLSDialer{}
	config := d.config("www.google.com", "443")
	if config.ServerName != "www.google.com" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"h2", "http/1.1"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigFromEmptyConfigForDoT(t *testing.T) {
	d := &TLSDialer{}
	config := d.config("dns.google", "853")
	if config.ServerName != "dns.google" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"dot"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigWithServerName(t *testing.T) {
	d := &TLSDialer{
		Config: &tls.Config{
			ServerName: "example.com",
		},
	}
	config := d.config("dns.google", "853")
	if config.ServerName != "example.com" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"dot"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigWithALPN(t *testing.T) {
	d := &TLSDialer{
		Config: &tls.Config{
			NextProtos: []string{"h2"},
		},
	}
	config := d.config("dns.google", "853")
	if config.ServerName != "dns.google" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"h2"}); diff != "" {
		t.Fatal(diff)
	}
}
