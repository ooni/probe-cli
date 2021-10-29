package filtering

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/google/martian/v3/mitm"
	"github.com/lucas-clemente/quic-go"
)

// newTLSConfig creates a new TLS config using google/martian/v3/mitm
func newTLSConfig() (*tls.Config, error) {
	cert, privkey, err := mitm.NewAuthority("jafar", "OONI", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	config, err := mitm.NewConfig(cert, privkey)
	if err != nil {
		return nil, err
	}
	return config.TLS(), nil
}

// MITMTLSProxy is a machine-in-the-middle TLS proxy.
type MITMTLSProxy struct{}

// Start starts the proxy.
func (p *MITMTLSProxy) Start(address string) (net.Listener, error) {
	config, err := newTLSConfig()
	if err != nil {
		return nil, err
	}
	listener, err := tls.Listen("tcp", address, config)
	if err != nil {
		return nil, err
	}
	go p.mainloop(listener)
	return listener, nil
}

func (p *MITMTLSProxy) mainloop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err == nil {
			go p.handle(conn)
			continue
		}
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			break
		}
	}
}

func (p *MITMTLSProxy) handle(conn net.Conn) {
	const timeout = 250 * time.Millisecond
	conn.SetDeadline(time.Now().Add(timeout))
	if tconn, good := conn.(*tls.Conn); good {
		tconn.Handshake()
	}
	conn.Close()
}

// MITMQUICProxy is a machine-in-the-middle QUIC proxy.
type MITMQUICProxy struct{}

// Start starts the proxy.
func (p *MITMQUICProxy) Start(address string) (quic.Listener, error) {
	config, err := newTLSConfig()
	if err != nil {
		return nil, err
	}
	config.NextProtos = []string{"h3"}
	listener, err := quic.ListenAddr(address, config, &quic.Config{})
	if err != nil {
		return nil, err
	}
	go p.mainloop(listener)
	return listener, nil
}

func (p *MITMQUICProxy) mainloop(listener quic.Listener) {
	for {
		sess, err := listener.Accept(context.Background())
		if err == nil {
			go p.handle(sess)
			continue
		}
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			break
		}
	}
}

func (p *MITMQUICProxy) handle(sess quic.Session) {
	sess.CloseWithError(0, "")
}
