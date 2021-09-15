package measure

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

// NewTLSConfigFromURL creates a new TLS config suitable
// to measure the specified URL. We will set the SNI from
// URL.Hostname(). We will use the Mozilla CA pool. We
// will configure the ALPN if the URL is "https" or "dot"
// otherwise we will leave the ALPN empty. The http3
// flag tells us whether we're configuring for TLS or QUIC.
//
// CAVEAT: when using parroting, part of the config may
// be ignored if it conflicts with the parroting.
func NewTLSConfigFromURL(URL *url.URL, http3 bool) *tls.Config {
	config := &tls.Config{
		ServerName: URL.Hostname(),
		RootCAs:    netxlite.NewDefaultCertPool(),
	}
	switch URL.Scheme {
	case "https":
		switch http3 {
		case true:
			config.NextProtos = []string{"h3"}
		default:
			config.NextProtos = []string{"h2", "http/1.1"}
		}
	case "dot":
		config.NextProtos = []string{"dot"}
	}
	return config
}

// TLSConn is a TLS conn.
type TLSConn = netxlite.TLSConn

// TLSHandshaker performs TLS handshakes.
type TLSHandshaker interface {
	TLSHandshake(
		ctx context.Context, tcpConn net.Conn, config *tls.Config) *TLSHandshakeResult
}

// TLSHandshakeResult is the result of TLSHandshake.
type TLSHandshakeResult struct {
	// Engine is the engine we have used.
	Engine string `json:"engine"`

	// Address is the remote endpoint address.
	Address string `json:"address"`

	// Config contains the TLS config.
	Config *TLSConfig `json:"config"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Failure contains the error (nil on success).
	Failure error `json:"failure"`

	// ConnectionState contains the connection state (set on success and
	// when the failure occurred during certificate verification).
	ConnectionState *TLSConnectionState `json:"connection_state"`

	// Conn is the TLS connection (set on success).
	Conn TLSConn `json:"-"`
}

// TLSConfig contains a subset of the TLS config.
type TLSConfig struct {
	// SNI is the SNI extension.
	SNI string `json:"sni"`

	// ALPN is the ALPN extension.
	ALPN []string `json:"alpn"`

	// NoTLSVerify indicates we disabled TLS verify.
	NoTLSVerify bool `json:"no_tls_verify"`
}

func newTLSConfig(in *tls.Config) *TLSConfig {
	return &TLSConfig{
		SNI:         in.ServerName,
		ALPN:        in.NextProtos,
		NoTLSVerify: in.InsecureSkipVerify,
	}
}

// TLSConnectionState contains a subset of the TLS connection state.
type TLSConnectionState struct {
	// TLSVersion is the TLS version.
	TLSVersion string `json:"tls_version"`

	// CipherSuite is the cipher suite.
	CipherSuite string `json:"cipher_suite"`

	// NegotiatedProtocol is the negotiated protocol.
	NegotiatedProtocol string `json:"negotiated_protocol"`

	// PeerCertificates contains the certificates.
	PeerCertificates [][]byte `json:"peer_certificates"`
}

func newTLSConnectionState(in *tls.ConnectionState) (out *TLSConnectionState) {
	out = &TLSConnectionState{
		TLSVersion:         netxlite.TLSVersionString(in.Version),
		CipherSuite:        netxlite.TLSCipherSuiteString(in.CipherSuite),
		NegotiatedProtocol: in.NegotiatedProtocol,
		PeerCertificates:   nil,
	}
	for _, cert := range in.PeerCertificates {
		out.PeerCertificates = append(out.PeerCertificates, cert.Raw)
	}
	return
}

// NewTLSHandshakerStdlib creates a new TLSHandshaker using
// the Go standard library to create connections.
func NewTLSHandshakerStdlib(begin time.Time, logger Logger) TLSHandshaker {
	return &tlsHandshaker{
		begin:  begin,
		engine: "stdlib",
		factory: func() netxlite.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(logger)
		},
	}
}

// NewTLSHandshakerUTLS creates a new TLSHandshaker using the
// Yawning fork of crypto/tls with the given parrot.
func NewTLSHandshakerUTLS(begin time.Time,
	logger Logger, parrot *utls.ClientHelloID) TLSHandshaker {
	return &tlsHandshaker{
		begin:  begin,
		engine: "yawning",
		factory: func() netxlite.TLSHandshaker {
			return netxlite.NewTLSHandshakerUTLS(logger, parrot)
		},
	}
}

type tlsHandshaker struct {
	begin   time.Time
	engine  string
	factory func() netxlite.TLSHandshaker
}

func (th *tlsHandshaker) TLSHandshake(
	ctx context.Context, tcpConn net.Conn, config *tls.Config) *TLSHandshakeResult {
	m := &TLSHandshakeResult{
		Engine:  th.engine,
		Address: tcpConn.RemoteAddr().String(),
		Config:  newTLSConfig(config),
		Started: time.Since(th.begin),
	}
	handshaker := th.factory()
	tlsConn, state, err := handshaker.Handshake(ctx, tcpConn, config)
	m.Completed = time.Since(th.begin)
	if err != nil {
		m.ConnectionState = newTLSConnectionStateFromError(err)
		m.Failure = err
		return m
	}
	m.ConnectionState = newTLSConnectionState(&state)
	m.Conn = tlsConn.(TLSConn) // Handshake guarantees this works
	return m
}

func newTLSConnectionStateFromError(err error) *TLSConnectionState {
	var (
		x509HostnameError           x509.HostnameError
		x509UnknownAuthorityError   x509.UnknownAuthorityError
		x509CertificateInvalidError x509.CertificateInvalidError
	)
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		return &TLSConnectionState{
			PeerCertificates: [][]byte{x509HostnameError.Certificate.Raw},
		}
	}
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		return &TLSConnectionState{
			PeerCertificates: [][]byte{x509UnknownAuthorityError.Cert.Raw},
		}
	}
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		return &TLSConnectionState{
			PeerCertificates: [][]byte{x509CertificateInvalidError.Cert.Raw},
		}
	}
	return nil
}
