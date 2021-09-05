package netxlite

import (
	"crypto/tls"
	"net"

	utls "gitlab.com/yawning/utls.git"
)

// UTLSConn implements TLSConn and uses a utls UConn as its underlying connection
type UTLSConn struct {
	*utls.UConn
}

// NewConnUTLS creates a NewConn function creating a utls connection with a specified ClientHelloID
func NewConnUTLS(clientHello *utls.ClientHelloID) func(conn net.Conn, config *tls.Config) TLSConn {
	return func(conn net.Conn, config *tls.Config) TLSConn {
		uConfig := &utls.Config{
			RootCAs:                     config.RootCAs,
			NextProtos:                  config.NextProtos,
			ServerName:                  config.ServerName,
			InsecureSkipVerify:          config.InsecureSkipVerify,
			DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		}
		tlsConn := utls.UClient(conn, uConfig, *clientHello)
		return &UTLSConn{tlsConn}
	}
}

func (c *UTLSConn) ConnectionState() tls.ConnectionState {
	uState := c.Conn.ConnectionState()
	return tls.ConnectionState{
		Version:                     uState.Version,
		HandshakeComplete:           uState.HandshakeComplete,
		DidResume:                   uState.DidResume,
		CipherSuite:                 uState.CipherSuite,
		NegotiatedProtocol:          uState.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  true,
		ServerName:                  uState.ServerName,
		PeerCertificates:            uState.PeerCertificates,
		VerifiedChains:              uState.VerifiedChains,
		SignedCertificateTimestamps: uState.SignedCertificateTimestamps,
		OCSPResponse:                uState.OCSPResponse,
		TLSUnique:                   uState.TLSUnique,
	}
}
