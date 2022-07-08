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
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/testingx"
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

func TestNewTLSHandshakerStdlib(t *testing.T) {
	th := NewTLSHandshakerStdlib(log.Log)
	logger := th.(*tlsHandshakerLogger)
	if logger.DebugLogger != log.Log {
		t.Fatal("invalid logger")
	}
	configurable := logger.TLSHandshaker.(*tlsHandshakerConfigurable)
	if configurable.NewConn != nil {
		t.Fatal("expected nil NewConn")
	}
}

func TestTLSHandshakerConfigurable(t *testing.T) {
	t.Run("Handshake", func(t *testing.T) {
		t.Run("with handshake I/O error", func(t *testing.T) {
			var times []time.Time
			h := &tlsHandshakerConfigurable{}
			tcpConn := &mocks.Conn{
				MockWrite: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockSetDeadline: func(t time.Time) error {
					times = append(times, t)
					return nil
				},
				MockRemoteAddr: func() net.Addr {
					return &mocks.Addr{
						MockString: func() string {
							return "1.1.1.1:443"
						},
						MockNetwork: func() string {
							return "tcp"
						},
					}
				},
			}
			ctx := context.Background()
			conn, state, err := h.Handshake(ctx, tcpConn, &tls.Config{
				ServerName: "x.org",
			})
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error that we expected")
			}
			var errWrapper *ErrWrapper
			if !errors.As(err, &errWrapper) {
				t.Fatal("the error has not been wrapped")
			}
			if errWrapper.Failure != FailureEOFError {
				t.Fatal("invalid wrapped error's failure")
			}
			if errWrapper.Operation != TLSHandshakeOperation {
				t.Fatal("invalid wrapped error's operation")
			}
			if !errors.Is(errWrapper.WrappedErr, io.EOF) {
				t.Fatal("invalid wrapped error's underlying error")
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
			if !reflect.ValueOf(state).IsZero() {
				t.Fatal("the returned connection state is not a zero value")
			}
		})

		t.Run("with success", func(t *testing.T) {
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
			handshaker := &tlsHandshakerConfigurable{}
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
		})

		t.Run("sets default root CA", func(t *testing.T) {
			expected := errors.New("mocked error")
			var gotTLSConfig *tls.Config
			handshaker := &tlsHandshakerConfigurable{
				NewConn: func(conn net.Conn, config *tls.Config) (TLSConn, error) {
					gotTLSConfig = config
					return &mocks.TLSConn{
						MockHandshakeContext: func(ctx context.Context) error {
							return expected
						},
					}, nil
				},
			}
			ctx := context.Background()
			config := &tls.Config{}
			conn := &mocks.Conn{
				MockSetDeadline: func(t time.Time) error {
					return nil
				},
				MockRemoteAddr: func() net.Addr {
					return &mocks.Addr{
						MockString: func() string {
							return "1.1.1.1:443"
						},
						MockNetwork: func() string {
							return "tcp"
						},
					}
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
		})

		t.Run("h.newConn fails", func(t *testing.T) {
			expected := errors.New("mocked error")
			handshaker := &tlsHandshakerConfigurable{
				NewConn: func(conn net.Conn, config *tls.Config) (TLSConn, error) {
					return nil, expected
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
		})

		t.Run("uses a context-injected custom trace (success case)", func(t *testing.T) {
			var (
				expectedSNI                 = "dns.google"
				goodStartStartTime          bool
				goodStartInsecureSkipVerify bool
				goodDoneInsecureSkipVerify  bool
				goodStartServerName         bool
				goodDoneServerName          bool
				goodDoneStartTime           bool
				goodDoneDoneTime            bool
				goodStartRemoteAddr         bool
				goodDoneRemoteAddr          bool
				goodDoneError               bool
				goodConnectionState         bool
				startCalled                 bool
				doneCalled                  bool
			)
			server := filtering.NewTLSServer(filtering.TLSActionBlockText)
			defer server.Close()
			zeroTime := time.Now()
			deterministicTime := testingx.NewTimeDeterministic(zeroTime)
			tx := &mocks.Trace{
				MockTimeNow: deterministicTime.Now,
				MockOnTLSHandshakeStart: func(now time.Time, remoteAddr string, config *tls.Config) {
					startCalled = true
					goodStartInsecureSkipVerify = (config.InsecureSkipVerify == true)
					goodStartServerName = (config.ServerName == expectedSNI)
					goodStartStartTime = (now.Sub(zeroTime) == 0)
					goodStartRemoteAddr = (remoteAddr == server.Endpoint())
				},
				MockOnTLSHandshakeDone: func(started time.Time, remoteAddr string, config *tls.Config, state tls.ConnectionState, err error, finished time.Time) {
					doneCalled = true
					goodDoneInsecureSkipVerify = (config.InsecureSkipVerify == true)
					goodDoneServerName = (config.ServerName == expectedSNI)
					goodDoneStartTime = (started.Sub(zeroTime) == 0)
					goodDoneDoneTime = (finished.Sub(zeroTime) == time.Second)
					goodDoneRemoteAddr = (remoteAddr == server.Endpoint())
					goodDoneError = (err == nil)
					goodConnectionState = (!reflect.ValueOf(state).IsZero())
				},
			}
			ctx := ContextWithTrace(context.Background(), tx)
			tcpConn, err := net.Dial("tcp", server.Endpoint())
			if err != nil {
				t.Fatal(err)
			}
			thx := NewTLSHandshakerStdlib(model.DiscardLogger)
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         expectedSNI,
			}
			tlsConn, connState, err := thx.Handshake(ctx, tcpConn, tlsConfig)
			if err != nil {
				t.Fatal(err)
			}
			tlsConn.Close()
			if reflect.ValueOf(connState).IsZero() {
				t.Fatal("expected nonzero connState")
			}
			if !startCalled {
				t.Fatal("start not called")
			}
			if !doneCalled {
				t.Fatal("done not called")
			}
			if !goodStartInsecureSkipVerify {
				t.Fatal("invalid start-event's InsecureSkipVerify")
			}
			if !goodDoneInsecureSkipVerify {
				t.Fatal("invalid done-event's InsecureSkipVerify")
			}
			if !goodStartServerName {
				t.Fatal("invalid start-event's ServerName")
			}
			if !goodDoneServerName {
				t.Fatal("invalid done-event's ServerName")
			}
			if !goodStartStartTime {
				t.Fatal("invalid start-event's start time")
			}
			if !goodDoneStartTime {
				t.Fatal("invalid done-event's start time")
			}
			if !goodDoneDoneTime {
				t.Fatal("invalid done-event's done time")
			}
			if !goodStartRemoteAddr {
				t.Fatal("invalid start-event's remoteAddr")
			}
			if !goodDoneRemoteAddr {
				t.Fatal("invalid done-event's remoteAddr")
			}
			if !goodDoneError {
				t.Fatal("invalid done-event's error")
			}
			if !goodConnectionState {
				t.Fatal("invalid done-event's connState")
			}
		})

		t.Run("uses a context-injected custom trace (failure case)", func(t *testing.T) {
			var (
				expectedEndpoint            = "8.8.8.8:443"
				expectedSNI                 = "dns.google"
				goodStartStartTime          bool
				goodStartInsecureSkipVerify bool
				goodDoneInsecureSkipVerify  bool
				goodStartServerName         bool
				goodDoneServerName          bool
				goodDoneStartTime           bool
				goodDoneDoneTime            bool
				goodStartRemoteAddr         bool
				goodDoneRemoteAddr          bool
				goodDoneError               bool
				goodConnectionState         bool
				startCalled                 bool
				doneCalled                  bool
			)
			zeroTime := time.Now()
			deterministicTime := testingx.NewTimeDeterministic(zeroTime)
			tx := &mocks.Trace{
				MockTimeNow: deterministicTime.Now,
				MockOnTLSHandshakeStart: func(now time.Time, remoteAddr string, config *tls.Config) {
					startCalled = true
					goodStartInsecureSkipVerify = (config.InsecureSkipVerify == true)
					goodStartServerName = (config.ServerName == expectedSNI)
					goodStartStartTime = (now.Sub(zeroTime) == 0)
					goodStartRemoteAddr = (remoteAddr == expectedEndpoint)
				},
				MockOnTLSHandshakeDone: func(started time.Time, remoteAddr string, config *tls.Config, state tls.ConnectionState, err error, finished time.Time) {
					doneCalled = true
					goodDoneInsecureSkipVerify = (config.InsecureSkipVerify == true)
					goodDoneServerName = (config.ServerName == expectedSNI)
					goodDoneStartTime = (started.Sub(zeroTime) == 0)
					goodDoneDoneTime = (finished.Sub(zeroTime) == time.Second)
					goodDoneRemoteAddr = (remoteAddr == expectedEndpoint)
					var ew *ErrWrapper
					goodDoneError = (errors.As(err, &ew) && ew.Error() == FailureEOFError)
					goodConnectionState = (reflect.ValueOf(state).IsZero())
				},
			}
			ctx := ContextWithTrace(context.Background(), tx)
			tcpConn := &mocks.Conn{
				MockSetDeadline: func(t time.Time) error {
					return nil
				},
				MockWrite: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockRemoteAddr: func() net.Addr {
					return &mocks.Addr{
						MockString: func() string {
							return expectedEndpoint
						},
						MockNetwork: func() string {
							return "tcp"
						},
					}
				},
			}
			thx := NewTLSHandshakerStdlib(model.DiscardLogger)
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         expectedSNI,
			}
			tlsConn, connState, err := thx.Handshake(ctx, tcpConn, tlsConfig)
			if !errors.Is(err, io.EOF) {
				t.Fatal("unexpected err", err)
			}
			if tlsConn != nil {
				t.Fatal("expected nil tlsConn")
			}
			if !reflect.ValueOf(connState).IsZero() {
				t.Fatal("expected zero connState")
			}
			if !startCalled {
				t.Fatal("start not called")
			}
			if !doneCalled {
				t.Fatal("done not called")
			}
			if !goodStartInsecureSkipVerify {
				t.Fatal("invalid start-event's InsecureSkipVerify")
			}
			if !goodDoneInsecureSkipVerify {
				t.Fatal("invalid done-event's InsecureSkipVerify")
			}
			if !goodStartServerName {
				t.Fatal("invalid start-event's ServerName")
			}
			if !goodDoneServerName {
				t.Fatal("invalid done-event's ServerName")
			}
			if !goodStartStartTime {
				t.Fatal("invalid start-event's start time")
			}
			if !goodDoneStartTime {
				t.Fatal("invalid done-event's start time")
			}
			if !goodDoneDoneTime {
				t.Fatal("invalid done-event's done time")
			}
			if !goodStartRemoteAddr {
				t.Fatal("invalid start-event's remoteAddr")
			}
			if !goodDoneRemoteAddr {
				t.Fatal("invalid done-event's remoteAddr")
			}
			if !goodDoneError {
				t.Fatal("invalid done-event's error")
			}
			if !goodConnectionState {
				t.Fatal("invalid done-event's connState")
			}
		})
	})
}

func TestTLSHandshakerLogger(t *testing.T) {
	t.Run("Handshake", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			th := &tlsHandshakerLogger{
				TLSHandshaker: &mocks.TLSHandshaker{
					MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
						return tls.Client(conn, config), tls.ConnectionState{}, nil
					},
				},
				DebugLogger: lo,
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
			if count != 2 {
				t.Fatal("invalid count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := errors.New("mocked error")
			th := &tlsHandshakerLogger{
				TLSHandshaker: &mocks.TLSHandshaker{
					MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
						return nil, tls.ConnectionState{}, expected
					},
				},
				DebugLogger: lo,
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
			if count != 2 {
				t.Fatal("invalid count")
			}
		})
	})
}

func TestNewTLSDialer(t *testing.T) {
	d := &mocks.Dialer{}
	th := &mocks.TLSHandshaker{}
	dialer := NewTLSDialer(d, th)
	tlsd := dialer.(*tlsDialer)
	if tlsd.Config == nil {
		t.Fatal("unexpected config")
	}
	if tlsd.Dialer != d {
		t.Fatal("unexpected dialer")
	}
	if tlsd.TLSHandshaker != th {
		t.Fatal("invalid handshaker")
	}
}

func TestTLSDialer(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		dialer := &tlsDialer{
			Dialer: &mocks.Dialer{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("DialTLSContext", func(t *testing.T) {
		t.Run("failure to split host and port", func(t *testing.T) {
			dialer := &tlsDialer{}
			ctx := context.Background()
			const address = "www.google.com" // missing port
			conn, err := dialer.DialTLSContext(ctx, "tcp", address)
			if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
				t.Fatal("not the error we expected", err)
			}
			if conn != nil {
				t.Fatal("connection is not nil")
			}
		})

		t.Run("failure dialing", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately fail
			dialer := tlsDialer{Dialer: &DialerSystem{}}
			conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
			if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
				t.Fatal("not the error we expected", err)
			}
			if conn != nil {
				t.Fatal("connection is not nil")
			}
		})

		t.Run("failure handshaking", func(t *testing.T) {
			ctx := context.Background()
			dialer := tlsDialer{
				Config: &tls.Config{},
				Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{MockWrite: func(b []byte) (int, error) {
						return 0, io.EOF
					}, MockClose: func() error {
						return nil
					}, MockSetDeadline: func(t time.Time) error {
						return nil
					}, MockRemoteAddr: func() net.Addr {
						return &mocks.Addr{
							MockNetwork: func() string {
								return "1.1.1.1:443"
							},
							MockString: func() string {
								return "tcp"
							},
						}
					}}, nil
				}},
				TLSHandshaker: &tlsHandshakerConfigurable{},
			}
			conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected", err)
			}
			if conn != nil {
				t.Fatal("connection is not nil")
			}
		})

		t.Run("success handshaking", func(t *testing.T) {
			ctx := context.Background()
			dialer := tlsDialer{
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
		})
	})

	t.Run("config", func(t *testing.T) {
		t.Run("from empty config for web", func(t *testing.T) {
			d := &tlsDialer{}
			config := d.config("www.google.com", "443")
			if config.ServerName != "www.google.com" {
				t.Fatal("invalid server name")
			}
			if diff := cmp.Diff(config.NextProtos, []string{"h2", "http/1.1"}); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("from empty config for dot", func(t *testing.T) {
			d := &tlsDialer{}
			config := d.config("dns.google", "853")
			if config.ServerName != "dns.google" {
				t.Fatal("invalid server name")
			}
			if diff := cmp.Diff(config.NextProtos, []string{"dot"}); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with server name", func(t *testing.T) {
			d := &tlsDialer{
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
		})

		t.Run("with alpn", func(t *testing.T) {
			d := &tlsDialer{
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
		})
	})
}

func TestNewSingleUseTLSDialer(t *testing.T) {
	conn := &mocks.TLSConn{}
	d := NewSingleUseTLSDialer(conn)
	defer d.CloseIdleConnections()
	outconn, err := d.DialTLSContext(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if conn != outconn {
		t.Fatal("invalid outconn")
	}
	for i := 0; i < 4; i++ {
		outconn, err = d.DialTLSContext(context.Background(), "", "")
		if !errors.Is(err, ErrNoConnReuse) {
			t.Fatal("not the error we expected", err)
		}
		if outconn != nil {
			t.Fatal("expected nil outconn here")
		}
	}
}

func TestNewNullTLSDialer(t *testing.T) {
	dialer := NewNullTLSDialer()
	conn, err := dialer.DialTLSContext(context.Background(), "", "")
	if !errors.Is(err, ErrNoTLSDialer) {
		t.Fatal("unexpected err", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
	dialer.CloseIdleConnections() // does not crash
}

func TestClonedTLSConfigOrNewEmptyConfig(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		var input *tls.Config
		output := ClonedTLSConfigOrNewEmptyConfig(input)
		if output == nil {
			t.Fatal("expected non-nil result")
		}
		v := reflect.ValueOf(*output)
		if !v.IsZero() {
			t.Fatal("expected zero config")
		}
	})

	t.Run("", func(t *testing.T) {
		input := &tls.Config{
			ServerName: "dns.google",
		}
		output := ClonedTLSConfigOrNewEmptyConfig(input)
		if output == input {
			t.Fatal("expected two distinct objects")
		}
		if !reflect.DeepEqual(input, output) {
			t.Fatal("apparently the two objects have different values")
		}
	})
}

func TestMaybeConnectionState(t *testing.T) {
	t.Run("with an error", func(t *testing.T) {
		returned := tls.ConnectionState{
			CipherSuite: tls.TLS_AES_128_GCM_SHA256,
		}
		conn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return returned
			},
		}
		state := tlsMaybeConnectionState(conn, errors.New("mocked error"))
		if !reflect.ValueOf(state).IsZero() {
			t.Fatal("expected to see a zero connection state")
		}
	})

	t.Run("without an error", func(t *testing.T) {
		returned := tls.ConnectionState{
			CipherSuite: tls.TLS_AES_128_GCM_SHA256,
		}
		conn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return returned
			},
		}
		state := tlsMaybeConnectionState(conn, nil)
		if reflect.ValueOf(state).IsZero() {
			t.Fatal("expected to see a nonzero connection state")
		}
	})
}
