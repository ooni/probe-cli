package netplumbing

// This file contains the implementation of UTLSHandshaker.

import (
	"context"
	"crypto/tls"
	"net"

	utls "gitlab.com/yawning/utls.git"
)

// UTLSHandshaker uses yawning/utls to perform the TLS handshake. To use
// this handshaker, create an instance and replace the default TLSHandshaker
// in the Config struct. Then used bind such a struct to your context.
type UTLSHandshaker struct {
	// ClientHelloID is the optional ClientHelloID. If not specified then
	// we use a suitable default value for the clientHelloID.
	ClientHelloID *utls.ClientHelloID
}

// TLSHandshake performs the TLS handshake using yawning/utls. We will
// honour selected fields of the original config and copy all the fields
// of the resulting state back to the *tls.ConnectionState.
func (th *UTLSHandshaker) TLSHandshake(
	ctx context.Context, tcpConn net.Conn,
	config *tls.Config) (net.Conn, *tls.ConnectionState, error) {
	// copy selected fields from the original config
	uConfig := &utls.Config{
		RootCAs:                     config.RootCAs,
		NextProtos:                  config.NextProtos,
		ServerName:                  config.ServerName,
		InsecureSkipVerify:          config.InsecureSkipVerify,
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
	}
	// perform the async handshake
	tlsConn := utls.UClient(tcpConn, uConfig, *th.clientHelloID(ctx))
	errch := make(chan error, 1)
	go func() { errch <- tlsConn.Handshake() }()
	select {
	case <-ctx.Done():
		// the context was interrupted during the handshake.
		return nil, nil, ctx.Err()
	case err := <-errch:
		if err != nil {
			return nil, nil, err
		}
		// fill the output from the original state
		uState := tlsConn.ConnectionState()
		state := &tls.ConnectionState{
			Version:                     uState.Version,
			HandshakeComplete:           uState.HandshakeComplete,
			DidResume:                   uState.DidResume,
			CipherSuite:                 uState.CipherSuite,
			NegotiatedProtocol:          uState.NegotiatedProtocol,
			ServerName:                  uState.ServerName,
			PeerCertificates:            uState.PeerCertificates,
			VerifiedChains:              uState.VerifiedChains,
			SignedCertificateTimestamps: uState.SignedCertificateTimestamps,
			OCSPResponse:                uState.OCSPResponse,
			TLSUnique:                   uState.TLSUnique,
		}
		return tlsConn, state, nil
	}
}

// clientHelloID returns the ClientHelloID we should use.
func (th *UTLSHandshaker) clientHelloID(ctx context.Context) *utls.ClientHelloID {
	if th.ClientHelloID != nil {
		return th.ClientHelloID
	}
	return &utls.HelloFirefox_Auto
}
