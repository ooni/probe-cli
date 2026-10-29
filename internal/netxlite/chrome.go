package netxlite

//
// TLS handshakes that parrot Chrome, using ooni/utls-light
//

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"reflect"

	"github.com/ooni/probe-cli/v3/internal/model"
	utlslight "github.com/ooni/utls-light/tls"
)

// NewTLSHandshakerChrome implements [model.MeasuringNetwork].
func (netx *Netx) NewTLSHandshakerChrome(logger model.DebugLogger) model.TLSHandshaker {
	return newTLSHandshakerLogger(&tlsHandshakerConfigurable{
		NewConn:  newChromeConn,
		provider: netx.MaybeCustomUnderlyingNetwork(),
	}, logger)
}

// ChromeConn implements TLSConn with utls-light, which sends Chrome's ClientHello.
type ChromeConn struct {
	*utlslight.Conn
	nc net.Conn
}

var _ TLSConn = &ChromeConn{}

// newChromeConn creates a new ChromeConn, accepting the same stdlib config
// fields as [NewUTLSConn].
func newChromeConn(conn net.Conn, config *tls.Config) (TLSConn, error) {
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
		if value.Field(idx).IsZero() || supportedFields[kind.Field(idx).Name] {
			continue
		}
		return nil, fmt.Errorf("%w: field %s is nonzero", errUTLSIncompatibleStdlibConfig, kind.Field(idx).Name)
	}
	lightConfig := &utlslight.Config{
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		InsecureSkipVerify:          config.InsecureSkipVerify, // #nosec G402 - the caller decides
		NextProtos:                  config.NextProtos,
		RootCAs:                     config.RootCAs,
		ServerName:                  config.ServerName,
	}
	return &ChromeConn{Conn: utlslight.Client(conn, lightConfig), nc: conn}, nil
}

// HandshakeContext implements TLSConn.
func (c *ChromeConn) HandshakeContext(ctx context.Context) error {
	return c.Conn.HandshakeContext(ctx)
}

// ConnectionState implements TLSConn.
func (c *ChromeConn) ConnectionState() tls.ConnectionState {
	s := c.Conn.ConnectionState()
	return tls.ConnectionState{
		Version:                     s.Version,
		HandshakeComplete:           s.HandshakeComplete,
		DidResume:                   s.DidResume,
		CipherSuite:                 s.CipherSuite,
		CurveID:                     tls.CurveID(s.CurveID),
		NegotiatedProtocol:          s.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  true,
		ServerName:                  s.ServerName,
		PeerCertificates:            s.PeerCertificates,
		VerifiedChains:              s.VerifiedChains,
		SignedCertificateTimestamps: s.SignedCertificateTimestamps,
		OCSPResponse:                s.OCSPResponse,
		TLSUnique:                   s.TLSUnique,
	}
}

// NetConn implements TLSConn.
func (c *ChromeConn) NetConn() net.Conn {
	return c.nc
}
