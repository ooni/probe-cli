package netemx

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go/http3"
)

// HTTP3ServerFactory implements [NetStackServerFactory] for HTTP-over-TLS (i.e., HTTPS).
//
// Use this factory along with [QAEnvOptionNetStack] to create HTTP3 servers.
type HTTP3ServerFactory struct {
	// Factory is the MANDATORY factory for creating the [http.Handler].
	Factory HTTPHandlerFactory

	// Ports is the MANDATORY list of ports where to listen.
	Ports []int

	// ServerNameMain is the MANDATORY server name we should configure.
	ServerNameMain string

	// ServerNameExtras contains OPTIONAL extra server names we should configure.
	ServerNameExtras []string
}

var _ NetStackServerFactory = &HTTP3ServerFactory{}

// MustNewServer implements NetStackServerFactory.
func (f *HTTP3ServerFactory) MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &http3Server{
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

type http3Server struct {
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
func (srv *http3Server) Close() error {
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
func (srv *http3Server) MustStart() {
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

func (srv *http3Server) mustListenPortLocked(handler http.Handler, ipAddr net.IP, port int) {
	// create the listening socket
	addr := &net.UDPAddr{IP: ipAddr, Port: port}
	listener := runtimex.Try1(srv.unet.ListenUDP("udp", addr))

	// create TLS config for the server name
	tlsConfig := srv.unet.MustNewServerTLSConfig(srv.serverNameMain, srv.serverNameExtras...)

	// serve requests in a background goroutine
	srvr := &http3.Server{
		TLSConfig: tlsConfig,
		Handler:   handler,
	}
	go srvr.Serve(listener)

	// For quic-go < 0.40.0, we needed the following fix as documented by the
	// https://github.com/ooni/probe/issues/2527 issue.
	//
	//	srv.closers = append(srv.closers, listener)
	//
	// After upgrading to quic-go/quic-go@v0.40.1 (https://github.com/ooni/probe-cli/pull/1428)
	// the above fix actually makes the code hang forever. We want to keep this message
	// around for a couple of cycles (say until ooni/probe-cli < 3.22).

	// make sure we track the server
	srv.closers = append(srv.closers, srvr)
}
