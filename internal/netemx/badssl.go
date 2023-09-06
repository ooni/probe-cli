package netemx

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// BadSSLServerFactory is a [NetStackServerFactory] that instantiates
// a [NetStackServer] used for testing common TLS issues.
type BadSSLServerFactory struct{}

var _ NetStackServerFactory = &BadSSLServerFactory{}

// MustNewServer implements NetStackServerFactory.
func (*BadSSLServerFactory) MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &badSSLServer{
		closers: []io.Closer{},
		logger:  env.Logger(),
		mu:      sync.Mutex{},
		unet:    stack,
	}
}

type badSSLServer struct {
	closers []io.Closer
	logger  model.Logger
	mu      sync.Mutex
	unet    *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *badSSLServer) Close() error {
	// "this method MUST be CONCURRENCY SAFE"
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// make sure we close all the child listeners
	for _, closer := range srv.closers {
		_ = closer.Close()
	}

	// "this method MUST be IDEMPOTENT"
	srv.closers = []io.Closer{}

	return nil
}

// MustStart implements NetStackServer.
func (srv *badSSLServer) MustStart() {
	// "this method MUST be CONCURRENCY SAFE"
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// build the listening endpoint
	ipAddr := net.ParseIP(srv.unet.IPAddress())
	runtimex.Assert(ipAddr != nil, "invalid IP address")
	epnt := &net.TCPAddr{IP: ipAddr, Port: 443}

	// start the server in a background goroutine
	server := testingx.MustNewTLSServerEx(epnt, srv.unet, &tlsHandlerForBadSSLServer{srv.unet})

	// track this listener as something to close later
	srv.closers = append(srv.closers, server)
}

// tlsHandlerForBadSSLServer is a [testingx.TLSHandler] that only cares about the
// handshake and applies specific wrong configurations during it.
//
// Limitation: this handler does not care about what happens after the handshake
// and just lets the underlying [*testingx.TLSServer] close the TLS conn.
type tlsHandlerForBadSSLServer struct {
	unet *netem.UNetStack
}

// GetCertificate implements testingx.TLSHandler.
func (thx *tlsHandlerForBadSSLServer) GetCertificate(
	ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	switch chi.ServerName {
	case "wrong.host.badssl.com":
		// Use the correct root CA but return a certificate for a different
		// host, which should cause the SNI verification to fail.
		tlsConfig := thx.unet.ServerTLSConfig()
		return tlsConfig.GetCertificate(&tls.ClientHelloInfo{
			CipherSuites:      chi.CipherSuites,
			ServerName:        "wrong-host.badssl.com", // different!
			SupportedCurves:   chi.SupportedCurves,
			SupportedPoints:   chi.SupportedPoints,
			SignatureSchemes:  chi.SignatureSchemes,
			SupportedProtos:   chi.SupportedProtos,
			SupportedVersions: chi.SupportedVersions,
			Conn:              tcpConn,
		})

	case "untrusted-root.badssl.com":
		fallthrough
	default:
		// Create a custom MITM config and use it to negotiate TLS. Because this would be
		// a different root CA, validating certs will fail the handshake.
		//
		// A more advanced version of this handler could choose different behaviors
		// depending on the SNI provided as part of the *tls.ClientHelloInfo.
		mitm := testingx.MustNewTLSMITMProviderNetem()
		tlsConfig := mitm.ServerTLSConfig()
		return tlsConfig.GetCertificate(chi)

	case "expired.badssl.com":
		// Create on-the-fly a certificate with the right SNI but that is clearly expired.
		mitm := thx.unet.TLSMITMConfig()
		return mitm.Config.NewCertWithoutCacheWithTimeNow(
			chi.ServerName,
			func() time.Time {
				return time.Date(2017, time.July, 17, 0, 0, 0, 0, time.UTC)
			},
		)
	}
}
