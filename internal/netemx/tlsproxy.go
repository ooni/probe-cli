package netemx

import (
	"io"
	"net"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// NewTLSProxyServerFactory is a [NetStackServerFactory] for the TCP echo service.
func NewTLSProxyServerFactory(logger model.Logger, ports ...uint16) NetStackServerFactory {
	return &tlsProxyServerFactory{
		logger: logger,
		ports:  ports,
	}
}

type tlsProxyServerFactory struct {
	logger model.Logger
	ports  []uint16
}

// MustNewServer implements NetStackServerFactory.
func (f *tlsProxyServerFactory) MustNewServer(_ NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return &tlsProxyServer{
		closers: []io.Closer{},
		logger:  f.logger,
		mu:      sync.Mutex{},
		ports:   f.ports,
		unet:    stack,
	}
}

type tlsProxyServer struct {
	closers []io.Closer
	logger  model.Logger
	mu      sync.Mutex
	ports   []uint16
	unet    *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *tlsProxyServer) Close() error {
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
func (srv *tlsProxyServer) MustStart() {
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

		server := testingx.MustNewTLSSNIProxyEx(
			srv.logger,
			&netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: srv.unet}},
			epnt,
			srv.unet,
		)

		// track this server as something to close later
		srv.closers = append(srv.closers, server)
	}
}
