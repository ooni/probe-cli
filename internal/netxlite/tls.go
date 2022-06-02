package netxlite

//
// TLS implementation
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TODO(bassosimone): check whether there's now equivalent functionality
// inside the standard library allowing us to map numbers to names.

var (
	tlsVersionString = map[uint16]string{
		tls.VersionTLS10: "TLSv1",
		tls.VersionTLS11: "TLSv1.1",
		tls.VersionTLS12: "TLSv1.2",
		tls.VersionTLS13: "TLSv1.3",
		0:                "", // guarantee correct behaviour
	}

	tlsCipherSuiteString = map[uint16]string{
		tls.TLS_RSA_WITH_RC4_128_SHA:                "TLS_RSA_WITH_RC4_128_SHA",
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA:            "TLS_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_RSA_WITH_AES_256_CBC_SHA:            "TLS_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256:         "TLS_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256:         "TLS_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384:         "TLS_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:    "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:  "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
		tls.TLS_AES_128_GCM_SHA256:                  "TLS_AES_128_GCM_SHA256",
		tls.TLS_AES_256_GCM_SHA384:                  "TLS_AES_256_GCM_SHA384",
		tls.TLS_CHACHA20_POLY1305_SHA256:            "TLS_CHACHA20_POLY1305_SHA256",
		0:                                           "", // guarantee correct behaviour
	}
)

// TLSVersionString returns a TLS version string. If value is zero, we
// return the empty string. If the value is unknown, we return
// `TLS_VERSION_UNKNOWN_ddd` where `ddd` is the numeric value passed
// to this function.
func TLSVersionString(value uint16) string {
	if str, found := tlsVersionString[value]; found {
		return str
	}
	return fmt.Sprintf("TLS_VERSION_UNKNOWN_%d", value)
}

// TLSCipherSuiteString returns the TLS cipher suite as a string. If value
// is zero, we return the empty string. If we don't know the mapping from
// the value to a cipher suite name, we return `TLS_CIPHER_SUITE_UNKNOWN_ddd`
// where `ddd` is the numeric value passed to this function.
func TLSCipherSuiteString(value uint16) string {
	if str, found := tlsCipherSuiteString[value]; found {
		return str
	}
	return fmt.Sprintf("TLS_CIPHER_SUITE_UNKNOWN_%d", value)
}

// NewDefaultCertPool returns the default x509 certificate pool
// that we bundle from Mozilla. It's safe to modify the returned
// value: every invocation returns a distinct *x509.CertPool instance.
func NewDefaultCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	// Assumption: AppendCertsFromPEM cannot fail because we
	// have a test in certify_test.go that guarantees that
	ok := pool.AppendCertsFromPEM([]byte(pemcerts))
	runtimex.PanicIfFalse(ok, "pool.AppendCertsFromPEM failed")
	return pool
}

// ErrInvalidTLSVersion indicates that you passed us a string
// that does not represent a valid TLS version.
var ErrInvalidTLSVersion = errors.New("invalid TLS version")

// ConfigureTLSVersion configures the correct TLS version into
// a *tls.Config or returns ErrInvalidTLSVersion.
//
// Recognized strings: TLSv1.3, TLSv1.2, TLSv1.1, TLSv1.0.
func ConfigureTLSVersion(config *tls.Config, version string) error {
	switch version {
	case "TLSv1.3":
		config.MinVersion = tls.VersionTLS13
		config.MaxVersion = tls.VersionTLS13
	case "TLSv1.2":
		config.MinVersion = tls.VersionTLS12
		config.MaxVersion = tls.VersionTLS12
	case "TLSv1.1":
		config.MinVersion = tls.VersionTLS11
		config.MaxVersion = tls.VersionTLS11
	case "TLSv1.0", "TLSv1":
		config.MinVersion = tls.VersionTLS10
		config.MaxVersion = tls.VersionTLS10
	case "":
		// nothing to do
	default:
		return ErrInvalidTLSVersion
	}
	return nil
}

// TLSConn is the type of connection that oohttp expects from
// any library that implements TLS functionality. By using this
// kind of TLSConn we're able to use both the standard library
// and gitlab.com/yawning/utls.git to perform TLS operations. Note
// that the stdlib's tls.Conn implements this interface.
type TLSConn = oohttp.TLSConn

// Ensures that a tls.Conn implements the TLSConn interface.
var _ TLSConn = &tls.Conn{}

// NewTLSHandshakerStdlib creates a new TLS handshaker using the
// go standard library to manage TLS.
//
// The handshaker guarantees:
//
// 1. logging
//
// 2. error wrapping
func NewTLSHandshakerStdlib(logger model.DebugLogger) model.TLSHandshaker {
	return newTLSHandshaker(&tlsHandshakerConfigurable{}, logger)
}

// newTLSHandshaker is the common factory for creating a new TLSHandshaker
func newTLSHandshaker(th model.TLSHandshaker, logger model.DebugLogger) model.TLSHandshaker {
	return &tlsHandshakerLogger{
		TLSHandshaker: &tlsHandshakerErrWrapper{
			TLSHandshaker: th,
		},
		DebugLogger: logger,
	}
}

// tlsHandshakerConfigurable is a configurable TLS handshaker that
// uses by default the standard library's TLS implementation.
type tlsHandshakerConfigurable struct {
	// NewConn is the OPTIONAL factory for creating a new connection. If
	// this factory is not set, we'll use the stdlib.
	NewConn func(conn net.Conn, config *tls.Config) (TLSConn, error)

	// Timeout is the OPTIONAL timeout imposed on the TLS handshake. If zero
	// or negative, we will use default timeout of 10 seconds.
	Timeout time.Duration
}

var _ model.TLSHandshaker = &tlsHandshakerConfigurable{}

// defaultCertPool is the cert pool we use by default. We store this
// value into a private variable to enable for unit testing.
var defaultCertPool = NewDefaultCertPool()

// Handshake implements Handshaker.Handshake. This function will
// configure the code to use the built-in Mozilla CA if the config
// field contains a nil RootCAs field.
func (h *tlsHandshakerConfigurable) Handshake(
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
	tlsconn, err := h.newConn(conn, config)
	if err != nil {
		return nil, tls.ConnectionState{}, err
	}
	if err := tlsconn.HandshakeContext(ctx); err != nil {
		return nil, tls.ConnectionState{}, err
	}
	return tlsconn, tlsconn.ConnectionState(), nil
}

// newConn creates a new TLSConn.
func (h *tlsHandshakerConfigurable) newConn(conn net.Conn, config *tls.Config) (TLSConn, error) {
	if h.NewConn != nil {
		return h.NewConn(conn, config)
	}
	// This used to be the place where we created a TLSConn using
	// github.com/ooni/oocrypto's TLS. However, it seems this strategy
	// does not correctly pick up the CPU capabilities. So, we have
	// now disabled oocrypto until we investigate, to avoid making the
	// development branch worse than it could in terms of TLS fingerprint.
	//
	// TODO(https://github.com/ooni/probe/issues/2122)
	return tls.Client(conn, config), nil
}

// defaultTLSHandshaker is the default TLS handshaker.
var defaultTLSHandshaker = &tlsHandshakerConfigurable{}

// tlsHandshakerLogger is a TLSHandshaker with logging.
type tlsHandshakerLogger struct {
	TLSHandshaker model.TLSHandshaker
	DebugLogger   model.DebugLogger
}

var _ model.TLSHandshaker = &tlsHandshakerLogger{}

// Handshake implements Handshaker.Handshake
func (h *tlsHandshakerLogger) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	h.DebugLogger.Debugf(
		"tls {sni=%s next=%+v}...", config.ServerName, config.NextProtos)
	start := time.Now()
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	elapsed := time.Since(start)
	if err != nil {
		h.DebugLogger.Debugf(
			"tls {sni=%s next=%+v}... %s in %s", config.ServerName,
			config.NextProtos, err, elapsed)
		return nil, tls.ConnectionState{}, err
	}
	h.DebugLogger.Debugf(
		"tls {sni=%s next=%+v}... ok in %s {next=%s cipher=%s v=%s}",
		config.ServerName, config.NextProtos, elapsed, state.NegotiatedProtocol,
		TLSCipherSuiteString(state.CipherSuite),
		TLSVersionString(state.Version))
	return tlsconn, state, nil
}

// NewTLSDialer creates a new TLS dialer using the given dialer and handshaker.
func NewTLSDialer(dialer model.Dialer, handshaker model.TLSHandshaker) model.TLSDialer {
	return NewTLSDialerWithConfig(dialer, handshaker, &tls.Config{})
}

// NewTLSDialerWithConfig is like NewTLSDialer with an optional config.
func NewTLSDialerWithConfig(d model.Dialer, h model.TLSHandshaker, c *tls.Config) model.TLSDialer {
	return &tlsDialer{Config: c, Dialer: d, TLSHandshaker: h}
}

// tlsDialer is the TLS dialer
type tlsDialer struct {
	// Config is the OPTIONAL tls config.
	Config *tls.Config

	// Dialer is the MANDATORY dialer.
	Dialer model.Dialer

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker model.TLSHandshaker
}

var _ model.TLSDialer = &tlsDialer{}

// CloseIdleConnections implements TLSDialer.CloseIdleConnections.
func (d *tlsDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// DialTLSContext implements TLSDialer.DialTLSContext.
func (d *tlsDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.config(host, port)
	tlsconn, _, err := d.TLSHandshaker.Handshake(ctx, conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return tlsconn, nil
}

// config creates a new config. If d.Config is nil, then we start
// from an empty config. Otherwise, we clone d.Config.
//
// We set the ServerName field if not already set.
//
// We set the ALPN if the port is 443 or 853, if not already set.
func (d *tlsDialer) config(host, port string) *tls.Config {
	config := d.Config
	if config == nil {
		config = &tls.Config{}
	}
	config = config.Clone() // operate on a clone
	if config.ServerName == "" {
		config.ServerName = host
	}
	if len(config.NextProtos) <= 0 {
		switch port {
		case "443":
			config.NextProtos = []string{"h2", "http/1.1"}
		case "853":
			config.NextProtos = []string{"dot"}
		}
	}
	return config
}

// NewSingleUseTLSDialer is like NewSingleUseDialer but takes
// in input a TLSConn rather than a net.Conn.
func NewSingleUseTLSDialer(conn TLSConn) model.TLSDialer {
	return &tlsDialerSingleUseAdapter{NewSingleUseDialer(conn)}
}

// tlsDialerSingleUseAdapter adapts dialerSingleUse to
// be a TLSDialer type rather than a Dialer type.
type tlsDialerSingleUseAdapter struct {
	Dialer model.Dialer
}

var _ model.TLSDialer = &tlsDialerSingleUseAdapter{}

// DialTLSContext implements TLSDialer.DialTLSContext.
func (d *tlsDialerSingleUseAdapter) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.Dialer.DialContext(ctx, network, address)
}

func (d *tlsDialerSingleUseAdapter) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// tlsHandshakerErrWrapper wraps the returned error to be an OONI error
type tlsHandshakerErrWrapper struct {
	TLSHandshaker model.TLSHandshaker
}

// Handshake implements TLSHandshaker.Handshake
func (h *tlsHandshakerErrWrapper) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	if err != nil {
		return nil, tls.ConnectionState{}, newErrWrapper(
			classifyTLSHandshakeError, TLSHandshakeOperation, err)
	}
	return tlsconn, state, nil
}

// ErrNoTLSDialer is the type of error returned by "null" TLS dialers
// when you attempt to dial with them.
var ErrNoTLSDialer = errors.New("no configured TLS dialer")

// NewNullTLSDialer returns a TLS dialer that always fails with ErrNoTLSDialer.
func NewNullTLSDialer() model.TLSDialer {
	return &nullTLSDialer{}
}

type nullTLSDialer struct{}

var _ model.TLSDialer = &nullTLSDialer{}

func (*nullTLSDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, ErrNoTLSDialer
}

func (*nullTLSDialer) CloseIdleConnections() {
	// nothing to do
}
