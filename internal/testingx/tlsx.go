package testingx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TLSMITMProvider provides TLS MITM capabilities. Two structs are known
// to implement this interface:
//
// 1. a [*netem.UNetStack] instance.
//
// 2. the one returned by [MustNewTLSMITMProviderNetem].
//
// Both use [github.com/google/martian/v3/mitm] under the hood.
//
// Use the former when you're using netem; the latter when using the stdlib.
type TLSMITMProvider interface {
	// DefaultCertPool returns the default cert pool to use.
	DefaultCertPool() (*x509.CertPool, error)

	// ServerTLSConfig returns ready to use server TLS configuration.
	ServerTLSConfig() *tls.Config
}

var _ TLSMITMProvider = &netem.UNetStack{}

// MustNewTLSMITMProviderNetem uses [github.com/ooni/netem] to implement [TLSMITMProvider].
func MustNewTLSMITMProviderNetem() TLSMITMProvider {
	return &netemTLSMITMProvider{runtimex.Try1(netem.NewTLSMITMConfig())}
}

type netemTLSMITMProvider struct {
	cfg *netem.TLSMITMConfig
}

// DefaultCertPool implements TLSMITMProvider.
func (p *netemTLSMITMProvider) DefaultCertPool() (*x509.CertPool, error) {
	return p.cfg.CertPool()
}

// ServerTLSConfig implements TLSMITMProvider.
func (p *netemTLSMITMProvider) ServerTLSConfig() *tls.Config {
	return p.cfg.TLSConfig()
}

// TLSHandler handles TLS connections. A handler should first handle the TLS handshake
// in the GetCertificate method. If GetCertificate did not return an error, and the
// handler implements [TLSConnHandler], its HandleTLSConn method will be called after
// the handshake to handle the lifecycle of the TLS conn itself.
type TLSHandler interface {
	// GetCertificate handles the TLS handshake.
	GetCertificate(ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error)
}

// TLSConn is the interface assumed by an established TLS conn.
type TLSConn interface {
	ConnectionState() tls.ConnectionState
	net.Conn
}

// TLSConnHandler is the interface implemented by handlers that want to handle
// and manage the established TLS connection after the handshake.
type TLSConnHandler interface {
	HandleTLSConn(conn TLSConn)
}

// TLSServer is a TLS server useful to implement test servers.
type TLSServer struct {
	// cancel unblocks background goroutines blocked on the context contolling their lifecycle.
	cancel context.CancelFunc

	// closeOnce provides "once" semantics when closing.
	closeOnce sync.Once

	// endpoint is the endpoint where we're listening.
	endpoint string

	// handler contains the TLSHandler.
	handler TLSHandler

	// listener is the listening socket controller.
	listener net.Listener
}

// MustNewTLSServer is a simplified [MustNewTLSServerEx] that uses the stdlib and localhost.
func MustNewTLSServer(handler TLSHandler) *TLSServer {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
	return MustNewTLSServerEx(addr, &TCPListenerStdlib{}, handler)
}

// MustNewTLSServerEx creates and starts a new TLSServer that executes
// the given action during the TLS handshake.
func MustNewTLSServerEx(addr *net.TCPAddr, tcpListener TCPListener, handler TLSHandler) *TLSServer {
	// create a listening socket
	listener := runtimex.Try1(tcpListener.ListenTCP("tcp", addr))

	// create context for interrupting goroutines blocked in the background
	ctx, cancel := context.WithCancel(context.Background())

	// create the server
	srv := &TLSServer{
		cancel:    cancel,
		closeOnce: sync.Once{},
		endpoint:  listener.Addr().String(),
		handler:   handler,
		listener:  listener,
	}

	// handle TCP connections
	go srv.mainloop(ctx)

	return srv
}

// Endpoint returns the endpoint where the server is listening.
func (p *TLSServer) Endpoint() string {
	return p.endpoint
}

// Close closes this server as soon as possible.
func (p *TLSServer) Close() (err error) {
	p.closeOnce.Do(func() {
		err = p.listener.Close()
		p.cancel()
	})
	return
}

func (p *TLSServer) mainloop(ctx context.Context) {
	for {
		conn, err := p.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			continue
		}

		// create a goroutine for connection, which is overkill in general
		// but reasonable for a server designed for testing
		go p.handle(ctx, conn)
	}
}

func (p *TLSServer) handle(ctx context.Context, tcpConn net.Conn) {
	// eventually close the TLS connection
	defer tcpConn.Close()

	// create TLS configuration where the handler is responsible for continuing the handshake
	tlsConfig := &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return p.handler.GetCertificate(ctx, tcpConn, chi)
		},
	}

	// create TLS connection
	tlsConn := tls.Server(tcpConn, tlsConfig)

	// perform the TLS handshake
	if err := tlsConn.Handshake(); err != nil {
		return
	}

	// eventually close the connection
	defer tlsConn.Close()

	// optionally let the handler handle the connection
	if h, good := p.handler.(TLSConnHandler); good {
		h.HandleTLSConn(tlsConn)
	}
}

// TLSHandlerTimeout returns a [TLSHandler] that reads data and never writes
// eventually causing the client connection to timeout.
func TLSHandlerTimeout() TLSHandler {
	return &tlsHandlerTimeout{}
}

type tlsHandlerTimeout struct{}

// GetCertificate implements TLSHandler.
func (*tlsHandlerTimeout) GetCertificate(
	ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	select {
	case <-time.After(300 * time.Second):
		return nil, context.DeadlineExceeded
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

const (
	// TLSAlertInternalError is the alter sent on internal errors
	TLSAlertInternalError = byte(80)

	// TLSAlertUnrecognizedName is the alert sent when the name is not recognized
	TLSAlertUnrecognizedName = byte(112)
)

// TLSHandlerSendAlert sends the alert given as argument to the client.
func TLSHandlerSendAlert(alert byte) TLSHandler {
	return &tlsHandlerSendAlert{alert}
}

type tlsHandlerSendAlert struct {
	alert byte
}

// GetCertificate implements TLSHandler.
func (thx *tlsHandlerSendAlert) GetCertificate(
	ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	alertdata := []byte{
		21, // alert
		3,  // version[0]
		3,  // version[1]
		0,  // length[0]
		2,  // length[1]
		2,  // fatal
		thx.alert,
	}
	_, _ = tcpConn.Write(alertdata)
	_ = tcpConn.Close()
	return nil, errors.New("internal error")
}

// TLSHandlerEOF closes the connection during the handshake.
func TLSHandlerEOF() TLSHandler {
	return &tlsHandlerEOF{}
}

type tlsHandlerEOF struct{}

// GetCertificate implements TLSHandler.
func (*tlsHandlerEOF) GetCertificate(ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	tcpConn.Close()
	return nil, errors.New("internal error")
}

// TLSHandlerReset resets the connection.
//
// Bug: this handler won't work with gvisor, which lacks TCPConn.SetLinger.
func TLSHandlerReset() TLSHandler {
	return &tlsHandlerReset{}
}

type tlsHandlerReset struct{}

// GetCertificate implements TLSHandler.
func (*tlsHandlerReset) GetCertificate(ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	tcpMaybeResetNetConn(tcpConn)
	tcpConn.Close() // just in case to avoid the error returned here to be sent remotely
	return nil, errors.New("internal error")
}

// TLSHandlerHandshakeAndWriteText returns a [TLSHandler] that attempts to
// complete the handshake and returns the given text to the caller.
func TLSHandlerHandshakeAndWriteText(mitm TLSMITMProvider, text []byte) TLSHandler {
	return &tlsHandlerHandshakeAndWriteText{mitm, text}
}

var _ TLSConnHandler = &tlsHandlerHandshakeAndWriteText{}

type tlsHandlerHandshakeAndWriteText struct {
	mitm TLSMITMProvider
	text []byte
}

// GetCertificate implements TLSHandler.
func (thx *tlsHandlerHandshakeAndWriteText) GetCertificate(ctx context.Context, tcpConn net.Conn, chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Implementation note: under the assumption that we're using github.com/ooni/netem in one way or
	// another here, the ServerTLSConfig method returns a suitable GetCertificate implementation. Since
	// the primary use case is that of using netem, this code is going to be WAI most of the times.
	config := thx.mitm.ServerTLSConfig()
	return config.GetCertificate(chi)
}

// HandleTLSConn implements TLSHandler.
func (thx *tlsHandlerHandshakeAndWriteText) HandleTLSConn(conn TLSConn) {
	_, _ = conn.Write(thx.text)
	// Note that the caller closes the connection for us
}
