package tracex

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestQUICDialerSaver(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {

		checkStartEventFields := func(t *testing.T, value *EventValue) {
			if value.Address != "8.8.8.8:443" {
				t.Fatal("invalid Address")
			}
			if !value.NoTLSVerify {
				t.Fatal("expected NoTLSVerify to be true")
			}
			if value.Proto != "udp" {
				t.Fatal("wrong protocol")
			}
			if diff := cmp.Diff(value.TLSNextProtos, []string{"h3"}); diff != "" {
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
			if _, good := ev.(*EventQUICHandshakeStart); !good {
				t.Fatal("invalid event type")
			}
			value := ev.Value()
			checkStartEventFields(t, value)
		}

		checkDoneEventFieldsSuccess := func(t *testing.T, value *EventValue) {
			if value.Duration <= 0 {
				t.Fatal("expected non-zero duration")
			}
			if value.Err.IsNotNil() {
				t.Fatal("expected no error here")
			}
			if value.TLSCipherSuite != "TLS_RSA_WITH_RC4_128_SHA" {
				t.Fatal("invalid cipher suite")
			}
			if value.TLSNegotiatedProto != "h3" {
				t.Fatal("invalid negotiated protocol")
			}
			if diff := cmp.Diff(value.TLSPeerCerts, [][]byte{{1, 2, 3, 4}}); diff != "" {
				t.Fatal(diff)
			}
			if value.TLSVersion != "TLSv1.3" {
				t.Fatal("invalid TLS version")
			}
		}

		checkDoneEvent := func(t *testing.T, ev Event, fun func(t *testing.T, value *EventValue)) {
			if _, good := ev.(*EventQUICHandshakeDone); !good {
				t.Fatal("invalid event type")
			}
			value := ev.Value()
			checkStartEventFields(t, value)
			fun(t, value)
		}

		t.Run("on success", func(t *testing.T) {
			saver := &Saver{}
			returnedConn := &mocks.QUICEarlyConnection{
				MockConnectionState: func() quic.ConnectionState {
					cs := quic.ConnectionState{}
					cs.TLS.ConnectionState.CipherSuite = tls.TLS_RSA_WITH_RC4_128_SHA
					cs.TLS.NegotiatedProtocol = "h3"
					cs.TLS.PeerCertificates = []*x509.Certificate{{
						Raw: []byte{1, 2, 3, 4},
					}}
					cs.TLS.Version = tls.VersionTLS13
					return cs
				},
			}
			dialer := saver.WrapQUICDialer(&mocks.QUICDialer{
				MockDialContext: func(ctx context.Context, address string,
					tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
					return returnedConn, nil
				},
			})
			ctx := context.Background()
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h3"},
				ServerName:         "dns.google",
			}
			quicConfig := &quic.Config{}
			conn, err := dialer.DialContext(ctx, "8.8.8.8:443", tlsConfig, quicConfig)
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
			if value.Err.IsNil() {
				t.Fatal("expected non-nil error here")
			}
		}

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			saver := &Saver{}
			dialer := saver.WrapQUICDialer(&mocks.QUICDialer{
				MockDialContext: func(ctx context.Context, address string,
					tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
					return nil, expected
				},
			})
			ctx := context.Background()
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h3"},
				ServerName:         "dns.google",
			}
			quicConfig := &quic.Config{}
			conn, err := dialer.DialContext(ctx, "8.8.8.8:443", tlsConfig, quicConfig)
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

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.QUICDialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		dialer := &QUICDialerSaver{
			QUICDialer: child,
			Saver:      &Saver{},
		}
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestWrapQUICListener(t *testing.T) {
	var saver *Saver
	ql := &mocks.QUICListener{}
	if saver.WrapQUICListener(ql) != ql {
		t.Fatal("unexpected result")
	}
}

func TestQUICListenerSaver(t *testing.T) {
	t.Run("on failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		saver := &Saver{}
		qls := saver.WrapQUICListener(&mocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expected
			},
		})
		pconn, err := qls.Listen(&net.UDPAddr{
			IP:   []byte{},
			Port: 8080,
			Zone: "",
		})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if pconn != nil {
			t.Fatal("expected nil pconn here")
		}
	})

	t.Run("on success", func(t *testing.T) {
		saver := &Saver{}
		returnedConn := &mocks.UDPLikeConn{}
		qls := saver.WrapQUICListener(&mocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return returnedConn, nil
			},
		})
		pconn, err := qls.Listen(&net.UDPAddr{
			IP:   []byte{},
			Port: 8080,
			Zone: "",
		})
		if err != nil {
			t.Fatal(err)
		}
		wconn := pconn.(*quicPacketConnWrapper)
		if wconn.UDPLikeConn != returnedConn {
			t.Fatal("invalid underlying connection")
		}
		if wconn.saver != saver {
			t.Fatal("invalid saver")
		}
	})
}

func TestQUICPacketConnWrapper(t *testing.T) {
	t.Run("ReadFrom", func(t *testing.T) {

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			saver := &Saver{}
			conn := &quicPacketConnWrapper{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockReadFrom: func(p []byte) (int, net.Addr, error) {
						return 0, nil, expected
					},
				},
				saver: saver,
			}
			buf := make([]byte, 1<<17)
			count, addr, err := conn.ReadFrom(buf)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("invalid count")
			}
			if addr != nil {
				t.Fatal("invalid addr")
			}
			events := saver.Read()
			if len(events) != 1 {
				t.Fatal("invalid number of events")
			}
			ev0 := events[0]
			if _, good := ev0.(*EventReadFromOperation); !good {
				t.Fatal("invalid event type")
			}
			value := ev0.Value()
			if value.Address != "" {
				t.Fatal("invalid Address")
			}
			if len(value.Data) != 0 {
				t.Fatal("invalid Data")
			}
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if value.Err != "unknown_failure: mocked error" {
				t.Fatal("unexpected value.Err", value.Err)
			}
			if value.NumBytes != 0 {
				t.Fatal("expected NumBytes")
			}
			if value.Time.IsZero() {
				t.Fatal("expected nonzero Time")
			}
		})

		t.Run("on success", func(t *testing.T) {
			expected := []byte{1, 2, 3, 4}
			saver := &Saver{}
			expectedAddr := &mocks.Addr{
				MockString: func() string {
					return "8.8.8.8:443"
				},
				MockNetwork: func() string {
					return "udp"
				},
			}
			conn := &quicPacketConnWrapper{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockReadFrom: func(p []byte) (int, net.Addr, error) {
						copy(p, expected)
						return len(expected), expectedAddr, nil
					},
				},
				saver: saver,
			}
			buf := make([]byte, 1<<17)
			count, addr, err := conn.ReadFrom(buf)
			if err != nil {
				t.Fatal(err)
			}
			if count != 4 {
				t.Fatal("invalid count")
			}
			if addr != expectedAddr {
				t.Fatal("invalid addr")
			}
			events := saver.Read()
			if len(events) != 1 {
				t.Fatal("invalid number of events")
			}
			ev0 := events[0]
			if _, good := ev0.(*EventReadFromOperation); !good {
				t.Fatal("invalid event type")
			}
			value := ev0.Value()
			if value.Address != "8.8.8.8:443" {
				t.Fatal("invalid Address")
			}
			if len(value.Data) != 4 {
				t.Fatal("invalid Data")
			}
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if value.Err.IsNotNil() {
				t.Fatal("unexpected value.Err", value.Err)
			}
			if value.NumBytes != 4 {
				t.Fatal("expected NumBytes")
			}
			if value.Time.IsZero() {
				t.Fatal("expected nonzero Time")
			}
		})
	})

	t.Run("WriteTo", func(t *testing.T) {

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			saver := &Saver{}
			conn := &quicPacketConnWrapper{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						return 0, expected
					},
				},
				saver: saver,
			}
			destAddr := &mocks.Addr{
				MockString: func() string {
					return "8.8.8.8:443"
				},
			}
			buf := make([]byte, 7)
			count, err := conn.WriteTo(buf, destAddr)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("invalid count")
			}
			events := saver.Read()
			if len(events) != 1 {
				t.Fatal("invalid number of events")
			}
			ev0 := events[0]
			if _, good := ev0.(*EventWriteToOperation); !good {
				t.Fatal("invalid event type")
			}
			value := ev0.Value()
			if value.Address != "8.8.8.8:443" {
				t.Fatal("invalid Address")
			}
			if len(value.Data) != 0 {
				t.Fatal("invalid Data")
			}
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if value.Err != "unknown_failure: mocked error" {
				t.Fatal("unexpected value.Err", value.Err)
			}
			if value.NumBytes != 0 {
				t.Fatal("expected NumBytes")
			}
			if value.Time.IsZero() {
				t.Fatal("expected nonzero Time")
			}
		})

		t.Run("on success", func(t *testing.T) {
			saver := &Saver{}
			conn := &quicPacketConnWrapper{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						return 1, nil
					},
				},
				saver: saver,
			}
			destAddr := &mocks.Addr{
				MockString: func() string {
					return "8.8.8.8:443"
				},
			}
			buf := make([]byte, 7)
			count, err := conn.WriteTo(buf, destAddr)
			if err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatal("invalid count")
			}
			events := saver.Read()
			if len(events) != 1 {
				t.Fatal("invalid number of events")
			}
			ev0 := events[0]
			if _, good := ev0.(*EventWriteToOperation); !good {
				t.Fatal("invalid event type")
			}
			value := ev0.Value()
			if value.Address != "8.8.8.8:443" {
				t.Fatal("invalid Address")
			}
			if len(value.Data) != 1 {
				t.Fatal("invalid Data")
			}
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if value.Err.IsNotNil() {
				t.Fatal("unexpected value.Err", value.Err)
			}
			if value.NumBytes != 1 {
				t.Fatal("expected NumBytes")
			}
			if value.Time.IsZero() {
				t.Fatal("expected nonzero Time")
			}
		})
	})
}
