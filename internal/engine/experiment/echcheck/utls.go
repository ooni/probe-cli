package echcheck

import (
	"context"
	"crypto/tls"
	"github.com/ooni/probe-cli/v3/internal/model"
	utls "gitlab.com/yawning/utls.git"
	"net"
)

type tlsHandshakerWithExtensions struct {
	conn       utlsConn
	extensions []utls.TLSExtension
	dl         model.DebugLogger
	id         *utls.ClientHelloID
}

var _ model.TLSHandshaker = &tlsHandshakerWithExtensions{}

// newHandshakerWithExtensions returns a NewHandsharer function for creating
// tlsHandshakerWithExtensions instances.
func newHandshakerWithExtensions(extensions []utls.TLSExtension) func(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
	return func(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
		return &tlsHandshakerWithExtensions{
			extensions: extensions,
			dl:         dl,
			id:         id,
		}
	}
}

func (t *tlsHandshakerWithExtensions) Handshake(ctx context.Context, conn net.Conn, tlsConfig *tls.Config) (
	net.Conn, tls.ConnectionState, error) {
	t.conn = newConnUTLSWithHelloID(conn, tlsConfig, t.id)

	if t.extensions != nil && len(t.extensions) != 0 {
		t.conn.BuildHandshakeState()
		t.conn.Extensions = append(t.conn.Extensions, t.extensions...)
	}

	err := t.conn.Handshake()

	return t.conn.NetConn(), t.conn.ConnectionState(), err
}

// utlsConn is a utls connection
type utlsConn struct {
	*utls.UConn
	nc net.Conn
}

// newConnUTLSWithHelloID creates a new connection with the given client hello ID.
func newConnUTLSWithHelloID(conn net.Conn, config *tls.Config, cid *utls.ClientHelloID) utlsConn {
	uConfig := &utls.Config{
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		InsecureSkipVerify:          config.InsecureSkipVerify,
		RootCAs:                     config.RootCAs,
		NextProtos:                  config.NextProtos,
		ServerName:                  config.ServerName,
	}
	tlsConn := utls.UClient(conn, uConfig, *cid)
	oconn := utlsConn{
		UConn: tlsConn,
		nc:    conn,
	}
	return oconn
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

func (c *utlsConn) NetConn() net.Conn {
	return c.nc
}
