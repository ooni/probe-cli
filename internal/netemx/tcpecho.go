package netemx

import (
	"io"
	"net"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewTCPEchoServerFactory is a [NetStackServerFactory] for the TCP echo service.
func NewTCPEchoServerFactory(logger model.Logger, ports ...uint16) NetStackServerFactory {
	return &tcpEchoServerFactory{
		logger: logger,
		ports:  ports,
	}
}

type tcpEchoServerFactory struct {
	logger model.Logger
	ports  []uint16
}

// MustNewServer implements NetStackServerFactory.
func (f *tcpEchoServerFactory) MustNewServer(_ NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &tcpEchoServer{
		closers: []io.Closer{},
		logger:  f.logger,
		mu:      sync.Mutex{},
		ports:   f.ports,
		unet:    stack,
	}
}

type tcpEchoServer struct {
	closers []io.Closer
	logger  model.Logger
	mu      sync.Mutex
	ports   []uint16
	unet    *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *tcpEchoServer) Close() error {
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
func (srv *tcpEchoServer) MustStart() {
	// "this method MUST be CONCURRENCY SAFE"
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// for each port of interest - note that here we panic liberally because we are
	// allowed to do so by the [NetStackServer] documentation.
	for _, port := range srv.ports {
		// create the endpoint address
		ipAddr := net.ParseIP(srv.unet.IPAddress())
		runtimex.Assert(ipAddr != nil, "invalid IP address")
		epnt := &net.TCPAddr{IP: ipAddr, Port: int(port)}

		// attempt to listen
		listener := runtimex.Try1(srv.unet.ListenTCP("tcp", epnt))

		// spawn goroutine for accepting
		go srv.acceptLoop(listener)

		// track this listener as something to close later
		srv.closers = append(srv.closers, listener)
	}
}

func (srv *tcpEchoServer) acceptLoop(listener net.Listener) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(srv.logger, "tcpEchoServer.acceptLoop")
	for {
		conn := runtimex.Try1(listener.Accept())
		go srv.serve(conn)
	}
}

func (srv *tcpEchoServer) serve(conn net.Conn) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(srv.logger, "tcpEchoServer.serve")

	// make sure we close the conn
	defer conn.Close()

	// loop until there is an I/O error
	for {
		buffer := make([]byte, 4096)
		count := runtimex.Try1(conn.Read(buffer))
		_, _ = conn.Write(buffer[:count])
	}
}
