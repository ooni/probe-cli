// Package stuntesting helps writing STUN tests.
package stuntesting

import (
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/pion/stun"
)

// Handler handles incoming STUN messages.
type Handler interface {
	Serve(req *stun.Message) (*stun.Message, error)
}

// HandlerFunc is an adapter that adapts an ordinary function to be a [Handler].
type HandlerFunc func(req *stun.Message) (*stun.Message, error)

var _ Handler = HandlerFunc(nil)

// Serve implements Handler
func (f HandlerFunc) Serve(req *stun.Message) (*stun.Message, error) {
	return f(req)
}

// ResponseWithAddPort generates a successful binding response with the given addr and port.
func ResponseWithAddPort(addr net.IP, port int) Handler {
	return HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
		resp := stun.MustBuild(stun.BindingSuccess)
		resp.TransactionID = req.TransactionID
		addr := &stun.XORMappedAddress{
			IP:   addr,
			Port: port,
		}
		runtimex.Try0(addr.AddTo(resp))
		return resp, nil
	})
}

// Server serves STUN requests
type Server struct {
	// address is the listening address
	address string

	// h is the [Handler] to use
	h Handler

	// once allows to call close just once
	once sync.Once

	// pconn is the listening UDP conn
	pconn net.PacketConn

	// wg is the wait group
	wg *sync.WaitGroup
}

// MustNewServer creates a new STUN [Server] or panics on error.
func MustNewServer(h Handler) *Server {
	pconn := runtimex.Try1(net.ListenPacket("udp", "127.0.0.1:0"))
	srv := &Server{
		address: pconn.LocalAddr().String(),
		h:       h,
		once:    sync.Once{},
		pconn:   pconn,
		wg:      &sync.WaitGroup{},
	}
	srv.wg.Add(1)
	go srv.serve()
	return srv
}

// Close shuts the server down. This method is idempotent.
func (s *Server) Close() (err error) {
	s.once.Do(func() {
		err = s.pconn.Close()
		s.wg.Wait()
	})
	return
}

// Address returns the server UDP address.
func (s *Server) Address() string {
	return s.address
}

// serveSTUN is a utility function that serves STUN requests
func (s *Server) serve() {
	// synchronize with close
	defer s.wg.Done()

	for {
		// read message from STUN client
		buffer := make([]byte, 1024)
		count, addr, err := s.pconn.ReadFrom(buffer)
		if err != nil {
			return
		}

		// parse message
		req := &stun.Message{
			Raw: buffer[:count],
		}
		if err := req.Decode(); err != nil {
			continue
		}

		// pass message to the handler
		resp, err := s.h.Serve(req)
		if err != nil {
			continue
		}

		// serialize message
		resp.Encode()
		_, _ = s.pconn.WriteTo(resp.Raw, addr)
	}
}
