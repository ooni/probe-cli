package netemx

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPSecureServerFactory implements [NetStackServerFactory] for HTTP-over-TLS (i.e., HTTPS).
//
// Use this factory along with [QAEnvOptionNetStack] to create HTTPS servers.
type HTTPSecureServerFactory struct {
	// Factory is the MANDATORY factory for creating the [http.Handler].
	Factory HTTPHandlerFactory

	// Ports is the MANDATORY list of ports where to listen.
	Ports []int

	// ServerNameMain is the MANDATORY server name we should configure.
	ServerNameMain string

	// ServerNameExtras contains OPTIONAL extra server names we should configure.
	ServerNameExtras []string
}

var _ NetStackServerFactory = &HTTPSecureServerFactory{}

// MustNewServer implements NetStackServerFactory.
func (f *HTTPSecureServerFactory) MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &httpSecureServer{
		closers:          []io.Closer{},
		env:              env,
		factory:          f.Factory,
		mu:               sync.Mutex{},
		ports:            f.Ports,
		serverNameMain:   f.ServerNameMain,
		serverNameExtras: f.ServerNameExtras,
		unet:             stack,
	}
}

type httpSecureServer struct {
	closers          []io.Closer
	env              NetStackServerFactoryEnv
	factory          HTTPHandlerFactory
	mu               sync.Mutex
	ports            []int
	serverNameMain   string
	serverNameExtras []string
	unet             *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *httpSecureServer) Close() error {
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
func (srv *httpSecureServer) MustStart() {
	// make the method locked as requested by the documentation
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// create the handler
	handler := srv.factory.NewHandler(srv.env, srv.unet)

	// create the listening address
	ipAddr := net.ParseIP(srv.unet.IPAddress())
	runtimex.Assert(ipAddr != nil, "expected valid IP address")

	for _, port := range srv.ports {
		srv.mustListenPortLocked(handler, ipAddr, port)
	}
}

func (srv *httpSecureServer) mustListenPortLocked(handler http.Handler, ipAddr net.IP, port int) {
	// create the listening socket
	addr := &net.TCPAddr{IP: ipAddr, Port: port}
	listener := runtimex.Try1(srv.unet.ListenTCP("tcp", addr))

	// create TLS config for the server name
	tlsConfig := srv.unet.MustNewServerTLSConfig(srv.serverNameMain, srv.serverNameExtras...)

	// serve requests in a background goroutine
	srvr := &http.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
	}
	go srvr.ServeTLS(listener, "", "")

	// make sure we track the server (the .Serve method will close the
	// listener once we close the server itself)
	srv.closers = append(srv.closers, srvr)
}
