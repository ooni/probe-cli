package netemx

import (
	"fmt"
	"io"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// UDPResolverFactory implements [NetStackServerFactory] for DNS-over-UDP servers.
//
// When this factory constructs a [NetStackServer], it will use:
//
// 1. the [NetStackServerFactoryEnv.OtherResolversConfig] as DNS configuration;
//
// 2. the [NetStackServerFactoryEnv.Logger] as the logger.
//
// Use this factory along with [QAEnvOptionNetStack] to create DNS-over-UDP servers.
type UDPResolverFactory struct{}

var _ NetStackServerFactory = &UDPResolverFactory{}

// MustNewServer implements NetStackServerFactory.
func (f *UDPResolverFactory) MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return udpResolverMustNewServer(env.OtherResolversConfig(), env.Logger(), stack)
}

type udpResolverFactoryForGetaddrinfo struct{}

var _ NetStackServerFactory = &udpResolverFactoryForGetaddrinfo{}

// MustNewServer implements NetStackServerFactory.
func (f *udpResolverFactoryForGetaddrinfo) MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer {
	return udpResolverMustNewServer(env.ISPResolverConfig(), env.Logger(), stack)
}

// udpResolverMustNewServer is an internal factory for creating a [NetStackServer] that
// runs a DNS-over-UDP server using the configured logger, DNS config, and stack.
func udpResolverMustNewServer(config *netem.DNSConfig, logger model.Logger, stack *netem.UNetStack) NetStackServer {
	return &udpResolver{
		closers: []io.Closer{},
		config:  config,
		logger:  logger,
		mu:      sync.Mutex{},
		unet:    stack,
	}
}

type udpResolver struct {
	closers []io.Closer
	config  *netem.DNSConfig
	logger  model.Logger
	mu      sync.Mutex
	unet    *netem.UNetStack
}

// Close implements NetStackServer.
func (srv *udpResolver) Close() error {
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
func (srv *udpResolver) MustStart() {
	// make the method locked as requested by the documentation
	defer srv.mu.Unlock()
	srv.mu.Lock()

	// Use a prefix logger for the DNS server
	prefixLogger := &logx.PrefixLogger{
		Prefix: fmt.Sprintf("%-16s", "RESOLVER"),
		Logger: srv.logger,
	}

	// create DNS server
	server := runtimex.Try1(netem.NewDNSServer(
		prefixLogger,
		srv.unet,
		srv.unet.IPAddress(),
		srv.config,
	))

	// track this closable
	srv.closers = append(srv.closers, server)
}
