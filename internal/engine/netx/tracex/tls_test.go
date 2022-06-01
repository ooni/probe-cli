package tracex

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestTLSHandshakerSaver(t *testing.T) {

	t.Run("Handshake", func(t *testing.T) {
		checkStartEventFields := func(t *testing.T, value *EventValue) {
			if value.Address != "8.8.8.8:443" {
				t.Fatal("invalid Address")
			}
			if !value.NoTLSVerify {
				t.Fatal("expected NoTLSVerify to be true")
			}
			if value.Proto != "tcp" {
				t.Fatal("wrong protocol")
			}
			if diff := cmp.Diff(value.TLSNextProtos, []string{"h2"}); diff != "" {
				t.Fatal(diff)
			}
			if value.TLSServerName != "dns.google" {
				t.Fatal("invalid TLSServerName")
			}
			if value.Time.IsZero() {
				t.Fatal("expected non zero time")
			}
		}

		checkStartedEvent := func(t *testing.T, ev Event) {
			if _, good := ev.(*EventTLSHandshakeStart); !good {
				t.Fatal("invalid event type")
			}
			value := ev.Value()
			checkStartEventFields(t, value)
		}

		checkDoneEventFieldsSuccess := func(t *testing.T, value *EventValue) {
			if value.Duration <= 0 {
				t.Fatal("expected non-zero duration")
			}
			if value.Err != nil {
				t.Fatal("expected no error here")
			}
			if value.TLSCipherSuite != "TLS_RSA_WITH_RC4_128_SHA" {
				t.Fatal("invalid cipher suite")
			}
			if value.TLSNegotiatedProto != "h2" {
				t.Fatal("invalid negotiated protocol")
			}
			if diff := cmp.Diff(value.TLSPeerCerts, []*x509.Certificate{}); diff != "" {
				t.Fatal(diff)
			}
			if value.TLSVersion != "TLSv1.3" {
				t.Fatal("invalid TLS version")
			}
		}

		checkDoneEvent := func(t *testing.T, ev Event, fun func(t *testing.T, value *EventValue)) {
			if _, good := ev.(*EventTLSHandshakeDone); !good {
				t.Fatal("invalid event type")
			}
			value := ev.Value()
			checkStartEventFields(t, value)
			fun(t, value)
		}

		t.Run("on success", func(t *testing.T) {
			saver := &Saver{}
			returnedConnState := tls.ConnectionState{
				CipherSuite:        tls.TLS_RSA_WITH_RC4_128_SHA,
				NegotiatedProtocol: "h2",
				PeerCertificates:   []*x509.Certificate{},
				Version:            tls.VersionTLS13,
			}
			returnedConn := &mocks.TLSConn{
				MockConnectionState: func() tls.ConnectionState {
					return returnedConnState
				},
			}
			thx := saver.WrapTLSHandshaker(&mocks.TLSHandshaker{
				MockHandshake: func(ctx context.Context, conn net.Conn,
					config *tls.Config) (net.Conn, tls.ConnectionState, error) {
					return returnedConn, returnedConnState, nil
				},
			})
			ctx := context.Background()
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h2"},
				ServerName:         "dns.google",
			}
			tcpConn := &mocks.Conn{
				MockRemoteAddr: func() net.Addr {
					return &mocks.Addr{
						MockString: func() string {
							return "8.8.8.8:443"
						},
						MockNetwork: func() string {
							return "tcp"
						},
					}
				},
			}
			conn, _, err := thx.Handshake(ctx, tcpConn, tlsConfig)
			if err != nil {
				t.Fatal(err)
			}
			if conn == nil {
				t.Fatal("expected non-nil conn")
			}
			events := saver.Read()
			if len(events) != 2 {
				t.Fatal("expected two events")
			}
			checkStartedEvent(t, events[0])
			checkDoneEvent(t, events[1], checkDoneEventFieldsSuccess)
		})

		checkDoneEventFieldsFailure := func(t *testing.T, value *EventValue) {
			if value.Duration <= 0 {
				t.Fatal("expected non-zero duration")
			}
			if value.Err == nil {
				t.Fatal("expected non-nil error here")
			}
			if value.TLSCipherSuite != "" {
				t.Fatal("invalid TLS cipher suite")
			}
			if value.TLSNegotiatedProto != "" {
				t.Fatal("invalid negotiated proto")
			}
			if len(value.TLSPeerCerts) > 0 {
				t.Fatal("expected no peer certs")
			}
			if value.TLSVersion != "" {
				t.Fatal("invalid TLS version")
			}
		}

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			saver := &Saver{}
			thx := saver.WrapTLSHandshaker(&mocks.TLSHandshaker{
				MockHandshake: func(ctx context.Context, conn net.Conn,
					config *tls.Config) (net.Conn, tls.ConnectionState, error) {
					return nil, tls.ConnectionState{}, expected
				},
			})
			ctx := context.Background()
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h2"},
				ServerName:         "dns.google",
			}
			tcpConn := &mocks.Conn{
				MockRemoteAddr: func() net.Addr {
					return &mocks.Addr{
						MockString: func() string {
							return "8.8.8.8:443"
						},
						MockNetwork: func() string {
							return "tcp"
						},
					}
				},
			}
			conn, _, err := thx.Handshake(ctx, tcpConn, tlsConfig)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
			events := saver.Read()
			if len(events) != 2 {
				t.Fatal("expected two events")
			}
			checkStartedEvent(t, events[0])
			checkDoneEvent(t, events[1], checkDoneEventFieldsFailure)
		})
	})
}

func Test_tlsPeerCerts(t *testing.T) {
	cert0 := &x509.Certificate{Raw: []byte{1, 2, 3, 4}}
	type args struct {
		state tls.ConnectionState
		err   error
	}
	tests := []struct {
		name string
		args args
		want []*x509.Certificate
	}{{
		name: "no error",
		args: args{
			state: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{cert0},
			},
		},
		want: []*x509.Certificate{cert0},
	}, {
		name: "all empty",
		args: args{},
		want: nil,
	}, {
		name: "x509.HostnameError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.HostnameError{
				Certificate: cert0,
			},
		},
		want: []*x509.Certificate{cert0},
	}, {
		name: "x509.UnknownAuthorityError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.UnknownAuthorityError{
				Cert: cert0,
			},
		},
		want: []*x509.Certificate{cert0},
	}, {
		name: "x509.CertificateInvalidError",
		args: args{
			state: tls.ConnectionState{},
			err: x509.CertificateInvalidError{
				Cert: cert0,
			},
		},
		want: []*x509.Certificate{cert0},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tlsPeerCerts(tt.args.state, tt.args.err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
