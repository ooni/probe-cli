package archival

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/fakefill"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverTLSHandshake(t *testing.T) {
	// newTLSHandshaker helps with building a TLS handshaker
	newTLSHandshaker := func(tlsConn net.Conn, state tls.ConnectionState, err error) model.TLSHandshaker {
		return &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, tcpConn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				time.Sleep(1 * time.Microsecond)
				return tlsConn, state, err
			},
		}
	}

	// newTCPConn creates a suitable net.Conn
	newTCPConn := func(address string) net.Conn {
		return &mocks.Conn{
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return address
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
			MockClose: func() error {
				return nil
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		var certs [][]byte
		ff := &fakefill.Filler{}
		ff.Fill(&certs)
		if len(certs) < 1 {
			t.Fatal("did not fill certs")
		}
		saver := NewSaver()
		v := &SingleQUICTLSHandshakeValidator{
			ExpectedALPN:       []string{"h2", "http/1.1"},
			ExpectedSNI:        "dns.google",
			ExpectedSkipVerify: true,
			//
			ExpectedCipherSuite:        tls.TLS_AES_128_GCM_SHA256,
			ExpectedNegotiatedProtocol: "h2",
			ExpectedPeerCerts:          certs,
			ExpectedVersion:            tls.VersionTLS12,
			//
			ExpectedNetwork:    "tcp",
			ExpectedRemoteAddr: mockedEndpoint,
			//
			QUICConfig:      nil, // this is not QUIC
			ExpectedFailure: nil,
			Saver:           saver,
		}
		expectedState := v.NewTLSConnectionState()
		thx := newTLSHandshaker(newTCPConn(mockedEndpoint), expectedState, nil)
		ctx := context.Background()
		tcpConn := newTCPConn(mockedEndpoint)
		conn, state, err := saver.TLSHandshake(ctx, thx, tcpConn, v.NewTLSConfig())
		if conn == nil {
			t.Fatal("expected non-nil conn")
		}
		conn.Close()
		if diff := cmp.Diff(expectedState, state, cmpopts.IgnoreUnexported(tls.ConnectionState{})); diff != "" {
			t.Fatal(diff)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	// failureFlow is the flow we run on failure.
	failureFlow := func(mockedError error, peerCerts [][]byte) error {
		const mockedEndpoint = "8.8.4.4:443"
		saver := NewSaver()
		v := &SingleQUICTLSHandshakeValidator{
			ExpectedALPN:       []string{"h2", "http/1.1"},
			ExpectedSNI:        "dns.google",
			ExpectedSkipVerify: true,
			//
			ExpectedCipherSuite:        0,
			ExpectedNegotiatedProtocol: "",
			ExpectedPeerCerts:          peerCerts,
			ExpectedVersion:            0,
			//
			ExpectedNetwork:    "tcp",
			ExpectedRemoteAddr: mockedEndpoint,
			//
			QUICConfig:      nil, // this is not QUIC
			ExpectedFailure: mockedError,
			Saver:           saver,
		}
		expectedState := v.NewTLSConnectionState()
		thx := newTLSHandshaker(nil, expectedState, mockedError)
		ctx := context.Background()
		tcpConn := newTCPConn(mockedEndpoint)
		conn, state, err := saver.TLSHandshake(ctx, thx, tcpConn, v.NewTLSConfig())
		if conn != nil {
			return errors.New("expected nil conn")
		}
		if diff := cmp.Diff(expectedState, state, cmpopts.IgnoreUnexported(tls.ConnectionState{})); diff != "" {
			return errors.New(diff)
		}
		if !errors.Is(err, mockedError) {
			return fmt.Errorf("unexpected err: %w", err)
		}
		return v.Validate()
	}

	t.Run("on generic failure", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		if err := failureFlow(mockedError, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on x509.HostnameError", func(t *testing.T) {
		var certificate []byte
		ff := &fakefill.Filler{}
		ff.Fill(&certificate)
		mockedError := x509.HostnameError{
			Certificate: &x509.Certificate{Raw: certificate},
		}
		if err := failureFlow(mockedError, [][]byte{certificate}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on x509.UnknownAuthorityError", func(t *testing.T) {
		var certificate []byte
		ff := &fakefill.Filler{}
		ff.Fill(&certificate)
		mockedError := x509.UnknownAuthorityError{
			Cert: &x509.Certificate{Raw: certificate},
		}
		if err := failureFlow(mockedError, [][]byte{certificate}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on x509.CertificateInvalidError", func(t *testing.T) {
		var certificate []byte
		ff := &fakefill.Filler{}
		ff.Fill(&certificate)
		mockedError := x509.CertificateInvalidError{
			Cert: &x509.Certificate{Raw: certificate},
		}
		if err := failureFlow(mockedError, [][]byte{certificate}); err != nil {
			t.Fatal(err)
		}
	})
}
