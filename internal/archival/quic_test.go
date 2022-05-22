package archival

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/marten-seemann/qtls-go1-18" // it's annoying to depend on that
	"github.com/ooni/probe-cli/v3/internal/fakefill"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverWriteTo(t *testing.T) {
	// newAddr creates an new net.Addr for testing.
	newAddr := func(endpoint string) net.Addr {
		return &mocks.Addr{
			MockString: func() string {
				return endpoint
			},
			MockNetwork: func() string {
				return "udp"
			},
		}
	}

	// newConn is a helper function for creating a new connection.
	newConn := func(numBytes int, err error) model.UDPLikeConn {
		return &mocks.UDPLikeConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				time.Sleep(time.Microsecond)
				return numBytes, err
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		const mockedNumBytes = 128
		addr := newAddr(mockedEndpoint)
		conn := newConn(mockedNumBytes, nil)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   mockedNumBytes,
			ExpectedErr:     nil,
			ExpectedNetwork: "udp",
			ExpectedOp:      netxlite.WriteToOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.WriteTo(conn, buf, addr)
		if err != nil {
			t.Fatal(err)
		}
		if count != mockedNumBytes {
			t.Fatal("invalid count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		addr := newAddr(mockedEndpoint)
		conn := newConn(0, mockedError)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     mockedError,
			ExpectedNetwork: "udp",
			ExpectedOp:      netxlite.WriteToOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.WriteTo(conn, buf, addr)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected err", err)
		}
		if count != 0 {
			t.Fatal("invalid count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSaverReadFrom(t *testing.T) {
	// newAddr creates an new net.Addr for testing.
	newAddr := func(endpoint string) net.Addr {
		return &mocks.Addr{
			MockString: func() string {
				return endpoint
			},
			MockNetwork: func() string {
				return "udp"
			},
		}
	}

	// newConn is a helper function for creating a new connection.
	newConn := func(numBytes int, addr net.Addr, err error) model.UDPLikeConn {
		return &mocks.UDPLikeConn{
			MockReadFrom: func(p []byte) (int, net.Addr, error) {
				time.Sleep(time.Microsecond)
				return numBytes, addr, err
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		const mockedNumBytes = 128
		expectedAddr := newAddr(mockedEndpoint)
		conn := newConn(mockedNumBytes, expectedAddr, nil)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   mockedNumBytes,
			ExpectedErr:     nil,
			ExpectedNetwork: "udp",
			ExpectedOp:      netxlite.ReadFromOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, addr, err := saver.ReadFrom(conn, buf)
		if err != nil {
			t.Fatal(err)
		}
		if expectedAddr.Network() != addr.Network() {
			t.Fatal("invalid addr.Network")
		}
		if expectedAddr.String() != addr.String() {
			t.Fatal("invalid addr.String")
		}
		if count != mockedNumBytes {
			t.Fatal("invalid count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		conn := newConn(0, nil, mockedError)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     mockedError,
			ExpectedNetwork: "udp",
			ExpectedOp:      netxlite.ReadFromOperation,
			ExpectedEpnt:    "",
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, addr, err := saver.ReadFrom(conn, buf)
		if !errors.Is(err, mockedError) {
			t.Fatal(err)
		}
		if addr != nil {
			t.Fatal("invalid addr")
		}
		if count != 0 {
			t.Fatal("invalid count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSaverQUICDialContext(t *testing.T) {
	// newQUICDialer creates a new QUICDialer for testing.
	newQUICDialer := func(qconn quic.EarlyConnection, err error) model.QUICDialer {
		return &mocks.QUICDialer{
			MockDialContext: func(
				ctx context.Context, network, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				time.Sleep(time.Microsecond)
				return qconn, err
			},
		}
	}

	// newQUICConnection creates a new quic.EarlyConnection for testing.
	newQUICConnection := func(handshakeComplete context.Context, state tls.ConnectionState) quic.EarlyConnection {
		return &mocks.QUICEarlyConnection{
			MockHandshakeComplete: func() context.Context {
				return handshakeComplete
			},
			MockConnectionState: func() quic.ConnectionState {
				return quic.ConnectionState{
					TLS: qtls.ConnectionStateWith0RTT{
						ConnectionState: state,
					},
				}
			},
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				return nil
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		handshakeCtx := context.Background()
		handshakeCtx, handshakeCancel := context.WithCancel(handshakeCtx)
		handshakeCancel() // simulate a completed handshake
		const expectedNetwork = "udp"
		const mockedEndpoint = "8.8.4.4:443"
		saver := NewSaver()
		var peerCerts [][]byte
		ff := &fakefill.Filler{}
		ff.Fill(&peerCerts)
		if len(peerCerts) < 1 {
			t.Fatal("did not fill peerCerts")
		}
		v := &SingleQUICTLSHandshakeValidator{
			ExpectedALPN:       []string{"h3"},
			ExpectedSNI:        "dns.google",
			ExpectedSkipVerify: true,
			//
			ExpectedCipherSuite:        tls.TLS_AES_128_GCM_SHA256,
			ExpectedNegotiatedProtocol: "h3",
			ExpectedPeerCerts:          peerCerts,
			ExpectedVersion:            tls.VersionTLS13,
			//
			ExpectedNetwork:    "quic",
			ExpectedRemoteAddr: mockedEndpoint,
			//
			QUICConfig: &quic.Config{},
			//
			ExpectedFailure: nil,
			Saver:           saver,
		}
		qconn := newQUICConnection(handshakeCtx, v.NewTLSConnectionState())
		dialer := newQUICDialer(qconn, nil)
		ctx := context.Background()
		qconn, err := saver.QUICDialContext(ctx, dialer, expectedNetwork,
			mockedEndpoint, v.NewTLSConfig(), v.QUICConfig)
		if err != nil {
			t.Fatal(err)
		}
		if qconn == nil {
			t.Fatal("expected nil qconn")
		}
		qconn.CloseWithError(0, "")
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on other error", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		const expectedNetwork = "udp"
		const mockedEndpoint = "8.8.4.4:443"
		saver := NewSaver()
		v := &SingleQUICTLSHandshakeValidator{
			ExpectedALPN:       []string{"h3"},
			ExpectedSNI:        "dns.google",
			ExpectedSkipVerify: true,
			//
			ExpectedCipherSuite:        0,
			ExpectedNegotiatedProtocol: "",
			ExpectedPeerCerts:          nil,
			ExpectedVersion:            0,
			//
			ExpectedNetwork:    "quic",
			ExpectedRemoteAddr: mockedEndpoint,
			//
			QUICConfig: &quic.Config{},
			//
			ExpectedFailure: mockedError,
			Saver:           saver,
		}
		dialer := newQUICDialer(nil, mockedError)
		ctx := context.Background()
		qconn, err := saver.QUICDialContext(ctx, dialer, expectedNetwork,
			mockedEndpoint, v.NewTLSConfig(), v.QUICConfig)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected error")
		}
		if qconn != nil {
			t.Fatal("expected nil connection")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	// TODO(bassosimone): here we're not testing the case in which
	// the certificate is invalid for the required SNI.
	//
	// We need first to figure out whether this is what happens
	// when we validate for QUIC in such cases. If that's the case
	// indeed, then we can write the tests.

	t.Run("on x509.HostnameError", func(t *testing.T) {
		t.Skip("test not implemented")
	})

	t.Run("on x509.UnknownAuthorityError", func(t *testing.T) {
		t.Skip("test not implemented")
	})

	t.Run("on x509.CertificateInvalidError", func(t *testing.T) {
		t.Skip("test not implemented")
	})
}

type SingleQUICTLSHandshakeValidator struct {
	// related to the tls.Config
	ExpectedALPN       []string
	ExpectedSNI        string
	ExpectedSkipVerify bool

	// related to the tls.ConnectionState
	ExpectedCipherSuite        uint16
	ExpectedNegotiatedProtocol string
	ExpectedPeerCerts          [][]byte
	ExpectedVersion            uint16

	// related to the mocked conn (TLS) / dial params (QUIC)
	ExpectedNetwork    string
	ExpectedRemoteAddr string

	// tells us whether we're using QUIC
	QUICConfig *quic.Config

	// other fields
	ExpectedFailure error
	Saver           *Saver
}

func (v *SingleQUICTLSHandshakeValidator) NewTLSConfig() *tls.Config {
	return &tls.Config{
		NextProtos:         v.ExpectedALPN,
		ServerName:         v.ExpectedSNI,
		InsecureSkipVerify: v.ExpectedSkipVerify,
	}
}

func (v *SingleQUICTLSHandshakeValidator) NewTLSConnectionState() tls.ConnectionState {
	var state tls.ConnectionState
	if v.ExpectedCipherSuite != 0 {
		state.CipherSuite = v.ExpectedCipherSuite
	}
	if v.ExpectedNegotiatedProtocol != "" {
		state.NegotiatedProtocol = v.ExpectedNegotiatedProtocol
	}
	for _, cert := range v.ExpectedPeerCerts {
		state.PeerCertificates = append(state.PeerCertificates, &x509.Certificate{
			Raw: cert,
		})
	}
	if v.ExpectedVersion != 0 {
		state.Version = v.ExpectedVersion
	}
	return state
}

func (v *SingleQUICTLSHandshakeValidator) Validate() error {
	trace := v.Saver.MoveOutTrace()
	var entries []*QUICTLSHandshakeEvent
	if v.QUICConfig != nil {
		entries = trace.QUICHandshake
	} else {
		entries = trace.TLSHandshake
	}
	if len(entries) != 1 {
		return errors.New("expected to see a single entry")
	}
	entry := entries[0]
	if diff := cmp.Diff(entry.ALPN, v.ExpectedALPN); diff != "" {
		return errors.New(diff)
	}
	if entry.CipherSuite != netxlite.TLSCipherSuiteString(v.ExpectedCipherSuite) {
		return errors.New("unexpected .CipherSuite")
	}
	if !errors.Is(entry.Failure, v.ExpectedFailure) {
		return errors.New("unexpected .Failure")
	}
	if !entry.Finished.After(entry.Started) {
		return errors.New(".Finished is not after .Started")
	}
	if entry.NegotiatedProto != v.ExpectedNegotiatedProtocol {
		return errors.New("unexpected .NegotiatedProto")
	}
	if entry.Network != v.ExpectedNetwork {
		return errors.New("unexpected .Network")
	}
	if diff := cmp.Diff(entry.PeerCerts, v.ExpectedPeerCerts); diff != "" {
		return errors.New("unexpected .PeerCerts")
	}
	if entry.RemoteAddr != v.ExpectedRemoteAddr {
		return errors.New("unexpected .RemoteAddr")
	}
	if entry.SNI != v.ExpectedSNI {
		return errors.New("unexpected .ServerName")
	}
	if entry.SkipVerify != v.ExpectedSkipVerify {
		return errors.New("unexpected .SkipVerify")
	}
	if entry.TLSVersion != netxlite.TLSVersionString(v.ExpectedVersion) {
		return errors.New("unexpected .Version")
	}
	return nil
}
