package measurexlite

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewTLSHandshakerStdlib(t *testing.T) {
	t.Run("NewTLSHandshakerStdlib creates a wrapped TLSHandshaker", func(t *testing.T) {
		underlying := &mocks.TLSHandshaker{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewTLSHandshakerStdlibFn = func(dl model.DebugLogger) model.TLSHandshaker {
			return underlying
		}
		thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
		thxt := thx.(*tlsHandshakerTrace)
		if thxt.thx != underlying {
			t.Fatal("invalid TLS handshaker")
		}
		if thxt.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("Handshake calls the underlying dialer with context-based tracing", func(t *testing.T) {
		expectedErr := errors.New("mocked err")
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		var hasCorrectTrace bool
		underlying := &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				gotTrace := netxlite.ContextTraceOrDefault(ctx)
				hasCorrectTrace = (gotTrace == trace)
				return nil, tls.ConnectionState{}, expectedErr
			},
		}
		trace.NewTLSHandshakerStdlibFn = func(dl model.DebugLogger) model.TLSHandshaker {
			return underlying
		}
		thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
		ctx := context.Background()
		conn, state, err := thx.Handshake(ctx, &mocks.Conn{}, &tls.Config{})
		if !errors.Is(err, expectedErr) {
			t.Fatal("unexpected err", err)
		}
		if !reflect.ValueOf(state).IsZero() {
			t.Fatal("expected zero-value state")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		if !hasCorrectTrace {
			t.Fatal("does not have the correct trace")
		}
	})

	t.Run("Handshake saves into the trace", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic timing
		thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
		ctx := context.Background()
		tcpConn := &mocks.Conn{
			MockSetDeadline: func(t time.Time) error {
				return nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
			MockWrite: func(b []byte) (int, error) {
				return 0, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "dns.cloudflare.com",
		}
		conn, state, err := thx.Handshake(ctx, tcpConn, tlsConfig)
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if !reflect.ValueOf(state).IsZero() {
			t.Fatal("expected zero-value state")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}

		t.Run("TLSHandshake events", func(t *testing.T) {
			events := trace.TLSHandshakes()
			if len(events) != 1 {
				t.Fatal("expected to see single TLSHandshake event")
			}
			expectedFailure := "unknown_failure: mocked"
			expect := &model.ArchivalTLSOrQUICHandshakeResult{
				Network:            "tls",
				Address:            "1.1.1.1:443",
				CipherSuite:        "",
				Failure:            &expectedFailure,
				NegotiatedProtocol: "",
				NoTLSVerify:        true,
				PeerCertificates:   []model.ArchivalMaybeBinaryData{},
				ServerName:         "dns.cloudflare.com",
				T:                  time.Second.Seconds(),
				Tags:               []string{},
				TLSVersion:         "",
			}
			got := events[0]
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 2 {
				t.Fatal("expected to see two Network events")
			}

			t.Run("tls_handshake_start", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "tls_handshake_start",
					Proto:     "",
					T:         0,
					Tags:      []string{},
				}
				got := events[0]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("tls_handshake_done", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "tls_handshake_done",
					Proto:     "",
					T:         time.Second.Seconds(),
					Tags:      []string{},
				}
				got := events[1]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})

	t.Run("Handshake discards events when buffers are full", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NetworkEvent = make(chan *model.ArchivalNetworkEvent)             // no buffer
		trace.TLSHandshake = make(chan *model.ArchivalTLSOrQUICHandshakeResult) // no buffer
		thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
		ctx := context.Background()
		tcpConn := &mocks.Conn{
			MockSetDeadline: func(t time.Time) error {
				return nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
			MockWrite: func(b []byte) (int, error) {
				return 0, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "dns.cloudflare.com",
		}
		conn, state, err := thx.Handshake(ctx, tcpConn, tlsConfig)
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if !reflect.ValueOf(state).IsZero() {
			t.Fatal("expected zero-value state")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}

		t.Run("TLSHandshake events", func(t *testing.T) {
			events := trace.TLSHandshakes()
			if len(events) != 0 {
				t.Fatal("expected to see no TLSHandshake events")
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 0 {
				t.Fatal("expected to see no Network events")
			}
		})
	})

	t.Run("we collect the desired data with a local TLS server", func(t *testing.T) {
		server := filtering.NewTLSServer(filtering.TLSActionBlockText)
		dialer := netxlite.NewDialerWithoutResolver(model.DiscardLogger)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "tcp", server.Endpoint())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		zeroTime := time.Now()
		dt := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = dt.Now // deterministic timing
		thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
		tlsConfig := &tls.Config{
			RootCAs:    server.CertPool(),
			ServerName: "dns.google",
		}
		tlsConn, connState, err := thx.Handshake(ctx, conn, tlsConfig)
		if err != nil {
			t.Fatal(err)
		}
		defer tlsConn.Close()
		data, err := netxlite.ReadAllContext(ctx, tlsConn)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, filtering.HTTPBlockpage451) {
			t.Fatal("bytes should match")
		}

		t.Run("TLSHandshake events", func(t *testing.T) {
			events := trace.TLSHandshakes()
			if len(events) != 1 {
				t.Fatal("expected to see a single TLSHandshake event")
			}
			expected := &model.ArchivalTLSOrQUICHandshakeResult{
				Network:            "tls",
				Address:            conn.RemoteAddr().String(),
				CipherSuite:        netxlite.TLSCipherSuiteString(connState.CipherSuite),
				Failure:            nil,
				NegotiatedProtocol: "",
				NoTLSVerify:        false,
				PeerCertificates:   []model.ArchivalMaybeBinaryData{},
				ServerName:         "dns.google",
				T:                  time.Second.Seconds(),
				Tags:               []string{},
				TLSVersion:         netxlite.TLSVersionString(connState.Version),
			}
			got := events[0]
			// TODO(bassosimone): it's still unclear to me how to test that
			// I am getting exactly the expected certificate here. I think the
			// certificate is generated on the fly by google/martian. So, I'm
			// just going to reduce the precision of this check.
			if len(got.PeerCertificates) != 2 {
				t.Fatal("expected to see two certificates")
			}
			got.PeerCertificates = []model.ArchivalMaybeBinaryData{} // see above
			if diff := cmp.Diff(expected, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 2 {
				t.Fatal("expected to see two Network events")
			}

			t.Run("tls_handshake_start", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "tls_handshake_start",
					Proto:     "",
					T:         0,
					Tags:      []string{},
				}
				got := events[0]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("tls_handshake_done", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "tls_handshake_done",
					Proto:     "",
					T:         time.Second.Seconds(),
					Tags:      []string{},
				}
				got := events[1]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})
}

func TestTLSPeerCerts(t *testing.T) {
	type args struct {
		state tls.ConnectionState
		err   error
	}
	tests := []struct {
		name    string
		args    args
		wantOut []model.ArchivalMaybeBinaryData
	}{{
		name: "x509.HostnameError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.HostnameError{
				Certificate: &x509.Certificate{
					Raw: []byte("deadbeef"),
				},
			},
		},
		wantOut: []model.ArchivalMaybeBinaryData{{
			Value: "deadbeef",
		}},
	}, {
		name: "x509.UnknownAuthorityError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.UnknownAuthorityError{
				Cert: &x509.Certificate{
					Raw: []byte("deadbeef"),
				},
			},
		},
		wantOut: []model.ArchivalMaybeBinaryData{{
			Value: "deadbeef",
		}},
	}, {
		name: "x509.CertificateInvalidError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.CertificateInvalidError{
				Cert: &x509.Certificate{
					Raw: []byte("deadbeef"),
				},
			},
		},
		wantOut: []model.ArchivalMaybeBinaryData{{
			Value: "deadbeef",
		}},
	}, {
		name: "successful case",
		args: args{
			state: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{{
					Raw: []byte("deadbeef"),
				}, {
					Raw: []byte("abad1dea"),
				}},
			},
			err: nil,
		},
		wantOut: []model.ArchivalMaybeBinaryData{{
			Value: "deadbeef",
		}, {
			Value: "abad1dea",
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := TLSPeerCerts(tt.args.state, tt.args.err)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
