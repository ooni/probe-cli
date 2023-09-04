package netemx

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPHandlerFactory constructs an [http.Handler].
type HTTPHandlerFactory interface {
	NewHandler() http.Handler
}

// HTTPHandlerFactoryFunc allows a func to become an [HTTPHandlerFactory].
type HTTPHandlerFactoryFunc func() http.Handler

var _ HTTPHandlerFactory = HTTPHandlerFactoryFunc(nil)

// NewHandler implements HTTPHandlerFactory.
func (fx HTTPHandlerFactoryFunc) NewHandler() http.Handler {
	return fx()
}

// HTTPCleartextServerFactory implements [NetStackServerFactory] for cleartext HTTP.
//
// Use this factory along with [QAEnvOptionNetStack] to create cleartext HTTP servers.
type HTTPCleartextServerFactory struct {
	// Factory is the MANDATORY factory for creating the [http.Handler].
	Factory HTTPHandlerFactory

	// Ports is the MANDATORY list of ports where to listen.
	Ports []int
}

var _ NetStackServerFactory = &HTTPCleartextServerFactory{}

// MustNewServer implements NetStackServerFactory.
func (f *HTTPCleartextServerFactory) MustNewServer(_ NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &httpCleartextServer{
		closers: []io.Closer{},
		factory: f.Factory,
		mu:      sync.Mutex{},
		ports:   f.Ports,
		unet:    stack,
	}
}

type httpCleartextServer struct {
	closers []io.Closer
	factory HTTPHandlerFactory
	mu      sync.Mutex
	ports   []int
	unet    *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *httpCleartextServer) Close() error {
	// make the method locked as requested by the documentation
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// close each of the closers
	for _, closer := range srv.closers {
		_ = closer.Close()
	}

	// be idempotent
	srv.closers = []io.Closer{}
	return nil
}

// MustStart implements NetStackServer.
func (srv *httpCleartextServer) MustStart() {
	// make the method locked as requested by the documentation
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// create the handler
	handler := srv.factory.NewHandler()

	// create the listening address
	ipAddr := net.ParseIP(srv.unet.IPAddress())
	runtimex.Assert(ipAddr != nil, "expected valid IP address")

	for _, port := range srv.ports {
		srv.mustListenPortLocked(handler, ipAddr, port)
	}
}

func (srv *httpCleartextServer) mustListenPortLocked(handler http.Handler, ipAddr net.IP, port int) {
	// create the listening socket
	addr := &net.TCPAddr{IP: ipAddr, Port: port}
	listener := runtimex.Try1(srv.unet.ListenTCP("tcp", addr))

	// serve requests in a background goroutine
	srvr := &http.Server{Handler: handler}
	go srvr.Serve(listener)

	// make sure we track the server (the .Serve method will close the
	// listener once we close the server itself)
	srv.closers = append(srv.closers, srvr)
}
