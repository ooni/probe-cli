package netxlite

import (
	"context"
	"crypto/tls"
	"net"

	utls "gitlab.com/yawning/utls.git"
)

// NewTLSHandshakerUTLS creates a new TLS handshaker using the
// gitlab.com/yawning/utls library to create TLS conns.
func NewTLSHandshakerUTLS(logger Logger, id *utls.ClientHelloID) TLSHandshaker {
	return &tlsHandshakerLogger{
		TLSHandshaker: &tlsHandshakerConfigurable{
			NewConn: newConnUTLS(id),
		},
		Logger: logger,
	}
}

// utlsConn implements TLSConn and uses a utls UConn as its underlying connection
type utlsConn struct {
	*utls.UConn
	testableHandshake func() error
}

// Ensures that a utlsConn implements the TLSConn interface.
var _ TLSConn = &utlsConn{}

// newConnUTLS returns a NewConn function for creating utlsConn instances.
func newConnUTLS(clientHello *utls.ClientHelloID) func(conn net.Conn, config *tls.Config) TLSConn {
	return func(conn net.Conn, config *tls.Config) TLSConn {
		uConfig := &utls.Config{
			RootCAs:                     config.RootCAs,
			NextProtos:                  config.NextProtos,
			ServerName:                  config.ServerName,
			InsecureSkipVerify:          config.InsecureSkipVerify,
			DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		}
		tlsConn := utls.UClient(conn, uConfig, *clientHello)
		return &utlsConn{UConn: tlsConn}
	}
}

func (c *utlsConn) HandshakeContext(ctx context.Context) error {
	errch := make(chan error, 1)
	go func() {
		errch <- c.handshakefn()()
	}()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *utlsConn) handshakefn() func() error {
	if c.testableHandshake != nil {
		return c.testableHandshake
	}
	return c.UConn.Handshake
}

func (c *utlsConn) ConnectionState() tls.ConnectionState {
	uState := c.Conn.ConnectionState()
	return tls.ConnectionState{
		Version:                     uState.Version,
		HandshakeComplete:           uState.HandshakeComplete,
		DidResume:                   uState.DidResume,
		CipherSuite:                 uState.CipherSuite,
		NegotiatedProtocol:          uState.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  uState.NegotiatedProtocolIsMutual,
		ServerName:                  uState.ServerName,
		PeerCertificates:            uState.PeerCertificates,
		VerifiedChains:              uState.VerifiedChains,
		SignedCertificateTimestamps: uState.SignedCertificateTimestamps,
		OCSPResponse:                uState.OCSPResponse,
		TLSUnique:                   uState.TLSUnique,
	}
}
