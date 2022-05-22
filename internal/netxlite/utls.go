package netxlite

//
// Code to use yawning/utls or refraction-networking/utls
//

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"

	"github.com/ooni/probe-cli/v3/internal/model"
	utls "gitlab.com/yawning/utls.git"
)

// NewTLSHandshakerUTLS creates a new TLS handshaker using
// gitlab.com/yawning/utls for TLS.
//
// The id is the address of something like utls.HelloFirefox_55.
//
// The handshaker guarantees:
//
// 1. logging
//
// 2. error wrapping
//
// Passing a nil `id` will make this function panic.
func NewTLSHandshakerUTLS(logger model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
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
func newConnUTLS(clientHello *utls.ClientHelloID) func(conn net.Conn, config *tls.Config) (TLSConn, error) {
	return func(conn net.Conn, config *tls.Config) (TLSConn, error) {
		return newConnUTLSWithHelloID(conn, config, clientHello)
	}
}

// errUTLSIncompatibleStdlibConfig indicates that the stdlib config you passed to
// newConnUTLSWithHelloID contains some fields we don't support.
var errUTLSIncompatibleStdlibConfig = errors.New("utls: incompatible stdlib config")

// newConnUTLSWithHelloID creates a new connection with the given client hello ID.
func newConnUTLSWithHelloID(conn net.Conn, config *tls.Config, cid *utls.ClientHelloID) (TLSConn, error) {
	supportedFields := map[string]bool{
		"DynamicRecordSizingDisabled": true,
		"InsecureSkipVerify":          true,
		"NextProtos":                  true,
		"RootCAs":                     true,
		"ServerName":                  true,
	}
	value := reflect.ValueOf(config).Elem()
	kind := value.Type()
	for idx := 0; idx < value.NumField(); idx++ {
		field := value.Field(idx)
		if field.IsZero() {
			continue
		}
		fieldKind := kind.Field(idx)
		if supportedFields[fieldKind.Name] {
			continue
		}
		err := fmt.Errorf("%w: field %s is nonzero", errUTLSIncompatibleStdlibConfig, fieldKind.Name)
		return nil, err
	}
	uConfig := &utls.Config{
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		InsecureSkipVerify:          config.InsecureSkipVerify,
		RootCAs:                     config.RootCAs,
		NextProtos:                  config.NextProtos,
		ServerName:                  config.ServerName,
	}
	tlsConn := utls.UClient(conn, uConfig, *cid)
	return &utlsConn{UConn: tlsConn}, nil
}

// ErrUTLSHandshakePanic indicates that there was panic handshaking
// when we were using the yawning/utls library for parroting.
// See https://github.com/ooni/probe/issues/1770 for more information.
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
