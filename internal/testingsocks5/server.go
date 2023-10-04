package testingsocks5

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	socks5Version = uint8(5)
)

// Server accepts connections and implements the SOCSK5 protocol.
//
// The zero value is invalid; please, use [NewServer].
type Server struct {
	// closeOnce ensures close has "once" semantics.
	closeOnce sync.Once

	// listener is the underlying listener.
	listener net.Listener

	// logger is the logger to use.
	logger model.Logger

	// netx is the network abstraction to use.
	netx *netxlite.Netx
}

// MustNewServer creates a new Server instance.
func MustNewServer(logger model.Logger, netx *netxlite.Netx, addr *net.TCPAddr) *Server {
	listener := runtimex.Try1(netx.ListenTCP("tcp", addr))
	server := &Server{
		closeOnce: sync.Once{},
		listener:  listener,
		logger: &logx.PrefixLogger{
			Prefix: "SOCKS5: ",
			Logger: logger,
		},
		netx: netx,
	}
	go server.Serve()
	return server
}

// Serve is used to Serve connections from a given listener.
func (s *Server) Serve() error {
	for {
		cconn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err := s.serveConn(cconn); err != nil {
				s.logger.Warnf("s.serveConn: %s", err.Error())
			}
		}()
	}
}

// serveConn is used to serve SOCKS5 over a single connection.
func (s *Server) serveConn(cconn net.Conn) error {
	defer cconn.Close()

	// Read the version byte
	version := []byte{0}
	if _, err := io.ReadFull(cconn, version); err != nil {
		return fmt.Errorf("failed to get version byte: %w", err)
	}

	s.logger.Infof("got version: %v", version)

	// Ensure we are compatible
	if version[0] != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %v", version)
	}

	// Authenticate the connection
	auth, err := s.authenticate(cconn)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	s.logger.Infof("authenticated: %+v", auth)

	request, err := newRequest(cconn)
	if err != nil {
		return fmt.Errorf("failed to read destination address: %w", err)
	}

	// Process the client request
	return s.handleRequest(request, cconn)
}

// Close closes the listener and waits for all goroutines to join
func (s *Server) Close() (err error) {
	s.closeOnce.Do(func() {
		err = s.listener.Close()
	})
	return
}

// Endpoint returns the server endpoint.
func (s *Server) Endpoint() string {
	return s.listener.Addr().String()
}

// URL returns a socks5 URL for the local listening address
func (s *Server) URL() *url.URL {
	return &url.URL{
		Scheme: "socks5",
		Host:   s.Endpoint(),
		Path:   "/",
	}
}
