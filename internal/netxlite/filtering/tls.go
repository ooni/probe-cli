package filtering

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSAction is a TLS filtering action that this proxy should take.
type TLSAction string

const (
	// TLSActionPass passes the traffic to the destination.
	TLSActionPass = TLSAction("pass")

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
)

// TLSProxy is a TLS proxy that routes the traffic depending
// on the SNI value and may implement filtering policies.
type TLSProxy struct {
	// OnIncomingSNI is the MANDATORY hook called whenever we have
	// successfully received a ClientHello message.
	OnIncomingSNI func(sni string) TLSAction
}

// Start starts the proxy.
func (p *TLSProxy) Start(address string) (net.Listener, error) {
	listener, _, err := p.start(address)
	return listener, err
}

func (p *TLSProxy) start(address string) (net.Listener, <-chan interface{}, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	done := make(chan interface{})
	go p.mainloop(listener, done)
	return listener, done, nil
}

func (p *TLSProxy) mainloop(listener net.Listener, done chan<- interface{}) {
	defer close(done)
	for p.oneloop(listener) {
		// nothing
	}
}

func (p *TLSProxy) oneloop(listener net.Listener) bool {
	conn, err := listener.Accept()
	if err != nil && strings.HasSuffix(err.Error(), "use of closed network connection") {
		return false // we need to stop
	}
	if err != nil {
		return true // we can continue running
	}
	go p.handle(conn)
	return true // we can continue running
}

const (
	tlsAlertInternalError    = byte(80)
	tlsAlertUnrecognizedName = byte(112)
)

func (p *TLSProxy) handle(conn net.Conn) {
	defer conn.Close()
	sni, hello, err := p.readClientHello(conn)
	if err != nil {
		p.reset(conn)
		return
	}
	switch p.OnIncomingSNI(sni) {
	case TLSActionPass:
		p.proxy(conn, sni, hello)
	case TLSActionTimeout:
		p.timeout(conn)
	case TLSActionAlertInternalError:
		p.alert(conn, tlsAlertInternalError)
	case TLSActionAlertUnrecognizedName:
		p.alert(conn, tlsAlertUnrecognizedName)
	case TLSActionEOF:
		p.eof(conn)
	default:
		p.reset(conn)
	}
}

// readClientHello reads the incoming ClientHello message.
//
// Arguments:
//
// - conn is the connection from which to read the ClientHello.
//
// Returns:
//
// - a string containing the SNI (empty on error);
//
// - bytes from the original ClientHello (nil on error);
//
// - an error (nil on success).
func (p *TLSProxy) readClientHello(conn net.Conn) (string, []byte, error) {
	connWrapper := &tlsClientHelloReader{Conn: conn}
	var (
		expectedErr = errors.New("cannot continue handhake")
		sni         string
		mutex       sync.Mutex // just for safety
	)
	err := tls.Server(connWrapper, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			mutex.Lock()
			sni = info.ServerName
			mutex.Unlock()
			return nil, expectedErr
		},
	}).Handshake()
	if !errors.Is(err, expectedErr) {
		return "", nil, err
	}
	return sni, connWrapper.clientHello, nil
}

// tlsClientHelloReader wraps a net.Conn for the purpose of
// saving the bytes of the ClientHello message.
type tlsClientHelloReader struct {
	net.Conn
	clientHello []byte
}

func (c *tlsClientHelloReader) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, err
	}
	c.clientHello = append(c.clientHello, b[:count]...)
	return count, nil
}

// Write prevents writing on the real connection
func (c *tlsClientHelloReader) Write(b []byte) (int, error) {
	return 0, errors.New("cannot write on this connection")
}

func (p *TLSProxy) reset(conn net.Conn) {
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetLinger(0)
	}
	conn.Close()
}

func (p *TLSProxy) timeout(conn net.Conn) {
	buffer := make([]byte, 1<<14)
	conn.Read(buffer)
	conn.Close()
}

func (p *TLSProxy) eof(conn net.Conn) {
	conn.Close()
}

func (p *TLSProxy) alert(conn net.Conn, code byte) {
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

func (p *TLSProxy) proxy(conn net.Conn, sni string, hello []byte) {
	p.proxydial(conn, sni, hello, net.Dial)
}

func (p *TLSProxy) proxydial(conn net.Conn, sni string, hello []byte,
	dial func(network, address string) (net.Conn, error)) {
	if sni == "" { // don't know the destination host
		p.reset(conn)
		return
	}
	serverconn, err := dial("tcp", net.JoinHostPort(sni, "443"))
	if err != nil {
		p.reset(conn)
		return
	}
	if p.connectingToMyself(serverconn) {
		p.reset(conn)
		return
	}
	if _, err := serverconn.Write(hello); err != nil {
		p.reset(conn)
		return
	}
	defer serverconn.Close() // conn is owned by the caller
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.forward(wg, conn, serverconn)
	go p.forward(wg, serverconn, conn)
	wg.Wait()
}

// connectingToMyself returns true when the proxy has been somehow
// forced to create a connection to itself.
func (p *TLSProxy) connectingToMyself(conn net.Conn) bool {
	local := conn.LocalAddr().String()
	localAddr, _, localErr := net.SplitHostPort(local)
	remote := conn.RemoteAddr().String()
	remoteAddr, _, remoteErr := net.SplitHostPort(remote)
	return localErr != nil || remoteErr != nil || localAddr == remoteAddr
}

// forward will forward the traffic.
func (p *TLSProxy) forward(wg *sync.WaitGroup, left net.Conn, right net.Conn) {
	defer wg.Done()
	netxlite.CopyContext(context.Background(), left, right)
}
