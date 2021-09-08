package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	utls "gitlab.com/yawning/utls.git"
)

// NewTLSHandshakerUTLS creates a new TLS handshaker using the
// gitlab.com/yawning/utls library to create TLS conns.
//
// The handshaker guarantees:
//
// 1. logging
//
// 2. error wrapping
//
// Passing a nil `id` will make this function panic.
func NewTLSHandshakerUTLS(logger Logger, id *utls.ClientHelloID) TLSHandshaker {
	return newTLSHandshaker(&tlsHandshakerConfigurable{
		NewConn: newConnUTLS(id),
	}, logger)
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

// ErrUTLSHandshakePanic indicates that there was panic handshaking
// when we were using the yawning/utls library for parroting.
//
// See https://github.com/ooni/probe/issues/1770
var ErrUTLSHandshakePanic = errors.New("utls: handshake panic")

func (c *utlsConn) HandshakeContext(ctx context.Context) (err error) {
	errch := make(chan error, 1)
	go func() {
		defer func() {
			// See https://github.com/ooni/probe/issues/1770
			if recover() != nil {
				errch <- ErrUTLSHandshakePanic
			}
		}()
		errch <- c.handshakefn()()
	}()
	select {
	case err = <-errch:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
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
