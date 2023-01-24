package filtering

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"time"

	"github.com/google/martian/v3/mitm"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TLSAction is a TLS filtering action that this proxy should take.
type TLSAction string

const (
	// TLSActionReset resets the connection.
	TLSActionReset = TLSAction("reset")

	// TLSActionTimeout causes the connection to timeout.
	TLSActionTimeout = TLSAction("timeout")

	// TLSActionEOF closes the connection.
	TLSActionEOF = TLSAction("eof")

	// TLSActionAlertInternalError sends an internal error
	// alert message to the TLS client.
	TLSActionAlertInternalError = TLSAction("internal-error")

	// TLSActionAlertUnrecognizedName tells the client that
	// it's handshaking with an unknown SNI.
	TLSActionAlertUnrecognizedName = TLSAction("alert-unrecognized-name")

	// TLSActionBlockText returns a static piece of text
	// to the client saying this website is blocked.
	TLSActionBlockText = TLSAction("block-text")
)

// TLSServer is a TLS server implementing filtering policies.
type TLSServer struct {
	// action is the action to perform.
	action TLSAction

	// cancel allows to cancel background operations.
	cancel context.CancelFunc

	// cert is the fake CA certificate.
	cert *x509.Certificate

	// config is the config to generate certificates on the fly.
	config *mitm.Config

	// done is closed when the background goroutine has terminated.
	done chan bool

	// endpoint is the endpoint where we're listening.
	endpoint string

	// listener is the TCP listener.
	listener net.Listener

	// privkey is the private key that signed the cert.
	privkey *rsa.PrivateKey
}

func tlsConfigMITM() (*x509.Certificate, *rsa.PrivateKey, *mitm.Config) {
	cert, privkey, err := mitm.NewAuthority("jafar", "OONI", 24*time.Hour)
	runtimex.PanicOnError(err, "mitm.NewAuthority failed")
	config, err := mitm.NewConfig(cert, privkey)
	runtimex.PanicOnError(err, "mitm.NewConfig failed")
	return cert, privkey, config
}

// NewTLSServer creates and starts a new TLSServer that executes
// the given action during the TLS handshake.
func NewTLSServer(action TLSAction) *TLSServer {
	done := make(chan bool)
	cert, privkey, config := tlsConfigMITM()
	listener := must.Listen("tcp", "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	endpoint := listener.Addr().String()
	server := &TLSServer{
		action:   action,
		cancel:   cancel,
		cert:     cert,
		config:   config,
		done:     done,
		endpoint: endpoint,
		listener: listener,
		privkey:  privkey,
	}
	go server.mainloop(ctx)
	return server
}

// CertPool returns the internal CA as a cert pool.
func (p *TLSServer) CertPool() *x509.CertPool {
	o := x509.NewCertPool()
	o.AddCert(p.cert)
	return o
}

// Endpoint returns the endpoint where the server is listening.
func (p *TLSServer) Endpoint() string {
	return p.endpoint
}

// Close closes this server as soon as possible.
func (p *TLSServer) Close() error {
	p.cancel()
	err := p.listener.Close()
	<-p.done
	return err
}

func (p *TLSServer) mainloop(ctx context.Context) {
	defer close(p.done)
	for p.oneloop(ctx) {
		// nothing
	}
}

func (p *TLSServer) oneloop(ctx context.Context) bool {
	conn, err := p.listener.Accept()
	if err != nil {
		return !errors.Is(err, net.ErrClosed)
	}
	go p.handle(ctx, conn)
	return true // we can continue running
}

const (
	tlsAlertInternalError    = byte(80)
	tlsAlertUnrecognizedName = byte(112)
)

func (p *TLSServer) handle(ctx context.Context, tcpConn net.Conn) {
	defer tcpConn.Close()
	tlsConn := tls.Server(tcpConn, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			switch p.action {
			case TLSActionTimeout:
				err := p.timeout(ctx, tcpConn)
				return nil, err
			case TLSActionAlertInternalError:
				p.alert(tcpConn, tlsAlertInternalError)
				return nil, errors.New("already sent alert")
			case TLSActionAlertUnrecognizedName:
				p.alert(tcpConn, tlsAlertUnrecognizedName)
				return nil, errors.New("already sent alert")
			case TLSActionEOF:
				p.eof(tcpConn)
				return nil, errors.New("already closed the connection")
			case TLSActionBlockText:
				return p.config.TLSForHost(info.ServerName).GetCertificate(info)
			default:
				p.reset(tcpConn)
				return nil, errors.New("already RST the connection")
			}
		},
	})
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	p.blockText(tlsConn)
	tlsConn.Close()
}

func (p *TLSServer) timeout(ctx context.Context, tcpConn net.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()
	<-ctx.Done()
	p.reset(tcpConn)
	return ctx.Err()
}

func (p *TLSServer) reset(conn net.Conn) {
	if tc, good := conn.(*net.TCPConn); good {
		tc.SetLinger(0)
	}
	conn.Close()
}

func (p *TLSServer) eof(conn net.Conn) {
	conn.Close()
}

func (p *TLSServer) alert(conn net.Conn, code byte) {
	alertdata := []byte{
		21, // alert
		3,  // version[0]
		3,  // version[1]
		0,  // length[0]
		2,  // length[1]
		2,  // fatal
		code,
	}
	conn.Write(alertdata)
	conn.Close()
}

func (p *TLSServer) blockText(tlsConn net.Conn) {
	tlsConn.Write(HTTPBlockpage451)
}
