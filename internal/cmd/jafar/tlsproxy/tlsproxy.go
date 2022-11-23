// Package tlsproxy contains a censoring TLS proxy. Most traffic is passed
// through using the SNI to choose the hostname to connect to. Specific offending
// SNIs are censored by returning a TLS alert to the client.
package tlsproxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/apex/log"
)

// Dialer establishes network connections
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// CensoringProxy is a censoring TLS proxy
type CensoringProxy struct {
	keywords     []string
	dial         func(network, address string) (net.Conn, error)
	outboundPort string
}

// NewCensoringProxy creates a new CensoringProxy instance using
// the specified list of keywords to censor. keywords is the list
// of keywords that trigger censorship if any of them appears in
// the SNI record of a ClientHello. dnsNetwork and dnsAddress are
// settings to configure the upstream, non censored DNS.
func NewCensoringProxy(
	keywords []string, uncensored Dialer, outboundPort string,
) *CensoringProxy {
	return &CensoringProxy{
		keywords: keywords,
		dial: func(network, address string) (net.Conn, error) {
			return uncensored.DialContext(context.Background(), network, address)
		},
		outboundPort: outboundPort,
	}
}

// handshakeReader is a hack to perform the initial part of the
// TLS handshake so to know the SNI and then replay the bytes of
// this initial part of the handshake with the server.
type handshakeReader struct {
	net.Conn
	incoming []byte
}

// Read saves the initial bytes of the handshake such that later
// we can replay the handshake with the real TLS server.
func (c *handshakeReader) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err == nil {
		c.incoming = append(c.incoming, b[:count]...)
	}
	return count, err
}

// Write prevents writing on the real connection
func (c *handshakeReader) Write(b []byte) (int, error) {
	return 0, errors.New("cannot write on this connection")
}

// forward forwards left traffic to right
func forward(wg *sync.WaitGroup, left, right net.Conn) {
	data := make([]byte, 1<<18)
	for {
		n, err := left.Read(data)
		if err != nil {
			break
		}
		if _, err = right.Write(data[:n]); err != nil {
			break
		}
	}
	wg.Done()
}

// reset closes the connection with a RST segment
func reset(conn net.Conn) {
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetLinger(0)
	}
	conn.Close()
}

// alertclose sends a TLS alert and then closes the connection
func alertclose(conn net.Conn) {
	alertdata := []byte{
		21, // alert
		3,  // version[0]
		3,  // version[1]
		0,  // length[0]
		2,  // length[1]
		2,  // fatal
		80, // internal error
	}
	conn.Write(alertdata)
	conn.Close()
}

// getsni attempts the handshakeReader hack to obtain the SNI by reading
// the beginning of the TLS handshake. On success a nonempty SNI string
// is returned. Otherwise we cannot distinguish between the absence of a
// SNI and any other reading network error that may have occurred.
func getsni(conn *handshakeReader) string {
	var (
		sni   string
		mutex sync.Mutex // just for safety
	)
	tls.Server(conn, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			mutex.Lock()
			sni = info.ServerName
			mutex.Unlock()
			return nil, errors.New("tlsproxy: we can't really continue handshake")
		},
	}).Handshake()
	return sni
}

func (p *CensoringProxy) connectingToMyself(conn net.Conn) bool {
	local := conn.LocalAddr().String()
	localAddr, _, localErr := net.SplitHostPort(local)
	remote := conn.RemoteAddr().String()
	remoteAddr, _, remoteErr := net.SplitHostPort(remote)
	return localErr != nil || remoteErr != nil || localAddr == remoteAddr
}

// handle implements the TLS SNI proxy
func (p *CensoringProxy) handle(clientconn net.Conn) {
	hr := &handshakeReader{Conn: clientconn}
	sni := getsni(hr)
	if sni == "" {
		log.Warn("tlsproxy: network failure or SNI not provided")
		reset(clientconn)
		return
	}
	for _, pattern := range p.keywords {
		if strings.Contains(sni, pattern) {
			log.Warnf("tlsproxy: reject SNI by policy: %s", sni)
			alertclose(clientconn)
			return
		}
	}
	serverconn, err := p.dial("tcp", net.JoinHostPort(sni, p.outboundPort))
	if err != nil {
		log.WithError(err).Warn("tlsproxy: p.dial failed")
		alertclose(clientconn)
		return
	}
	if p.connectingToMyself(serverconn) {
		log.Warn("tlsproxy: connecting to myself")
		alertclose(clientconn)
		return
	}
	if _, err := serverconn.Write(hr.incoming); err != nil {
		log.WithError(err).Warn("tlsproxy: serverconn.Write failed")
		alertclose(clientconn)
		return
	}
	log.Debugf("tlsproxy: routing for %s", sni)
	defer clientconn.Close()
	defer serverconn.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go forward(&wg, clientconn, serverconn)
	go forward(&wg, serverconn, clientconn)
	wg.Wait()
}

func (p *CensoringProxy) run(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil && strings.Contains(
			err.Error(), "use of closed network connection") {
			return
		}
		if err == nil {
			// It's difficult to make accept fail, so restructure
			// the code such that we enter into the happy path
			go p.handle(conn)
		}
	}
}

// Start starts the censoring proxy.
func (p *CensoringProxy) Start(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	go p.run(listener)
	return listener, nil
}
