package netxlite

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	utls "gitlab.com/yawning/utls.git"
)

// TLSHandshaker is the generic TLS handshaker.
type TLSHandshaker interface {
	// Handshake creates a new TLS connection from the given connection and
	// the given config. This function DOES NOT take ownership of the connection
	// and it's your responsibility to close it on failure.
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// TLSHandshakerConfigurable is a configurable TLS handshaker that
// uses by default the standard library's TLS implementation.
type TLSHandshakerConfigurable struct {
	// NewConn is the OPTIONAL factory for creating a new connection. If
	// this factory is not set, we'll use the stdlib.
	NewConn func(conn net.Conn, config *tls.Config) TLSConn

	// Timeout is the OPTIONAL timeout imposed on the TLS handshake. If zero
	// or negative, we will use default timeout of 10 seconds.
	Timeout time.Duration
}

var _ TLSHandshaker = &TLSHandshakerConfigurable{}

// defaultCertPool is the cert pool we use by default. We store this
// value into a private variable to enable for unit testing.
var defaultCertPool = NewDefaultCertPool()

// Handshake implements Handshaker.Handshake. This function will
// configure the code to use the built-in Mozilla CA if the config
// field contains a nil RootCAs field.
//
// Bug
//
// Until Go 1.17 is released, this function will not honour
// the context. We'll however always enforce an overall timeout.
func (h *TLSHandshakerConfigurable) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	defer conn.SetDeadline(time.Time{})
	conn.SetDeadline(time.Now().Add(timeout))
	if config.RootCAs == nil {
		config = config.Clone()
		config.RootCAs = defaultCertPool
	}
	tlsconn := h.newConn(conn, config)
	if err := tlsconn.Handshake(); err != nil {
		return nil, tls.ConnectionState{}, err
	}
	return tlsconn, tlsconn.ConnectionState(), nil
}

// newConn creates a new TLSConn.
func (h *TLSHandshakerConfigurable) newConn(conn net.Conn, config *tls.Config) TLSConn {
	if h.NewConn != nil {
		return h.NewConn(conn, config)
	}
	return tls.Client(conn, config)
}

// DefaultTLSHandshaker is the default TLS handshaker.
var DefaultTLSHandshaker = &TLSHandshakerConfigurable{}

// TLSHandshakerLogger is a TLSHandshaker with logging.
type TLSHandshakerLogger struct {
	// TLSHandshaker is the underlying handshaker.
	TLSHandshaker TLSHandshaker

	// Logger is the underlying logger.
	Logger Logger
}

// Handshake implements Handshaker.Handshake
func (h *TLSHandshakerLogger) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	h.Logger.Debugf(
		"tls {sni=%s next=%+v}...", config.ServerName, config.NextProtos)
	start := time.Now()
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	elapsed := time.Since(start)
	if err != nil {
		h.Logger.Debugf(
			"tls {sni=%s next=%+v}... %s in %s", config.ServerName,
			config.NextProtos, err, elapsed)
		return nil, tls.ConnectionState{}, err
	}
	h.Logger.Debugf(
		"tls {sni=%s next=%+v}... ok in %s {next=%s cipher=%s v=%s}",
		config.ServerName, config.NextProtos, elapsed, state.NegotiatedProtocol,
		TLSCipherSuiteString(state.CipherSuite),
		TLSVersionString(state.Version))
	return tlsconn, state, nil
}

// UTLSConn implements TLSConn and uses a utls UConn as its underlying connection
type UTLSConn struct {
	*utls.UConn
}

// NewConnUTLS creates a NewConn function creating a utls connection with a specified ClientHelloID
func NewConnUTLS(clientHello string) func(conn net.Conn, config *tls.Config) TLSConn {
	var clientHelloID *utls.ClientHelloID
	switch strings.ToLower(clientHello) {
	case "chrome":
		clientHelloID = &utls.HelloChrome_Auto
	case "firefox":
		clientHelloID = &utls.HelloFirefox_Auto
	case "ios":
		clientHelloID = &utls.HelloIOS_Auto
	case "golang":
		clientHelloID = &utls.HelloGolang
	default:
		clientHelloID = &utls.HelloChrome_Auto
	}
	return func(conn net.Conn, config *tls.Config) TLSConn {
		uConfig := &utls.Config{
			RootCAs:                     config.RootCAs,
			NextProtos:                  config.NextProtos,
			ServerName:                  config.ServerName,
			InsecureSkipVerify:          config.InsecureSkipVerify,
			DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
		}
		tlsConn := utls.UClient(conn, uConfig, *clientHelloID)
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
		ServerName:                  uState.ServerName,
		PeerCertificates:            uState.PeerCertificates,
		VerifiedChains:              uState.VerifiedChains,
		SignedCertificateTimestamps: uState.SignedCertificateTimestamps,
		OCSPResponse:                uState.OCSPResponse,
		TLSUnique:                   uState.TLSUnique,
	}
}

var _ TLSHandshaker = &TLSHandshakerLogger{}
