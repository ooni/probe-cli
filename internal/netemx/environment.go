package netemx

import (
	"io"
	"net"
	"net/http"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go/http3"
)

const (
	// DefaultClientAddress is the address used by default for a client.
	DefaultClientAddress = "10.0.0.14"

	// DefaultClientResolver is the resolver used by default by client.
	DefaultClientResolver = "10.0.0.1"

	// DefaultServersResolver is the the resolver used by default by server.
	DefaultServersResolver = "1.1.1.1"
)

// The netemx environment design is based on netemx_test.
// TODO(kelmenhorst): consider writing netemx_test.go using this Environment.

// Environment is a configurable [netem] QA environment with a DNS server
// stack, multiple server stacks, and a client stack. The zero value is not
// ready to use. You should use [NewEnvironment] to construct.
type Environment struct {
	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// dpi refers to the [netem.DPIEngine] we're using.
	dpi *netem.DPIEngine

	// topology is the topology we're using.
	topology *netem.StarTopology

	// closables contains all entities where we have to take care of closing
	closables []io.Closer
}

// ClientConfig configures the client in the Environment.
type ClientConfig struct {
	// ClientAddr is the OPTIONAL address of the client stack.
	// If empty, we use DefaultClientAddress.
	ClientAddr string

	// DNSConfig is the MANDATORY [*netem.DNSConfig] to be used for the DNS in this environment.
	DNSConfig *netem.DNSConfig

	// ResolverAddr is the OPTIONAL address of the default resolver of the client to
	// be used in the environment. If empty, we use DefaultClientResolver.
	ResolverAddr string
}

// ServersConfig configures the servers in the Environment.
type ServersConfig struct {
	// DNSConfig is the MANDATORY [*netem.DNSConfig] to be used for the DNS in this environment.
	DNSConfig *netem.DNSConfig

	// ResolverAddr is the OPTIONAL address of the default resolver to be used in the environment.
	// If empty, we use DefaultServerResolver.
	ResolverAddr string

	// Servers is the MANDATORY list of [ServerStack]s to be used in this environment.
	Servers []ConfigServerStack
}

// ConfigServerStack represents a server instance.
// Multiple HTTP servers can run on the same server, on different ports.
type ConfigServerStack struct {
	// ServerAddr is the MANDATORY address of the web server stack.
	ServerAddr string

	// HTTPServers is the MANDATORY list of [HTTPServer], i.e. server instances on this stack.
	HTTPServers []ConfigHTTPServer
}

// ConfigHTTPServer is a handler running on a server port. A ConfigHTTPServer
// might use QUIC instead of TCP as transport.
type ConfigHTTPServer struct {
	// Port is the MANDATORY port that this HTTP server is running on.
	Port int

	// QUIC indicates whether this HTTP server uses QUIC instead of TCP as transport.
	QUIC bool

	// Handler OPTIONALLY specifies the handler to use for this HTTP server.
	Handler http.Handler
}

// configureClient creates the client network stack
func configureClient(
	clientConfig *ClientConfig,
	topology *netem.StarTopology,
	dpi *netem.DPIEngine,
) (*netem.UNetStack, *netem.DNSServer) {
	// set the default resolver address
	resolverAddr := clientConfig.ResolverAddr
	if resolverAddr == "" {
		resolverAddr = DefaultClientResolver
	}

	// create dns server stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	dnsServerStack := runtimex.Try1(topology.AddHost(
		resolverAddr, // server IP address
		resolverAddr, // default resolver address
		&netem.LinkConfig{
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))

	// create DNS server using the dnsServerStack
	dnsServer := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		dnsServerStack,
		resolverAddr,
		clientConfig.DNSConfig,
	))

	// create client stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	//
	// TODO(bassosimone,kelmenhorst): consider allowing to configure the
	// delays and losses should the need for this arise in the future.
	clientStack := runtimex.Try1(topology.AddHost(
		DefaultClientAddress,
		resolverAddr,
		&netem.LinkConfig{
			DPIEngine:        dpi,
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))

	return clientStack, dnsServer
}

// configureServer creates a single server network stack by creating and launching HTTP(3) servers
func configureServer(
	s *ConfigServerStack,
	topology *netem.StarTopology,
	resolverAddr string,
) []io.Closer {

	// create server stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	serverStack := runtimex.Try1(topology.AddHost(
		s.ServerAddr,
		resolverAddr,
		&netem.LinkConfig{
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))

	// create the array of closables, i.e. HTTP(3) servers and UDP sockets, to return
	var closables []io.Closer

	// configure and start HTTP server instances running on the server stack
	for _, l := range s.HTTPServers {
		// make sure there is a handler
		handler := l.Handler
		if handler == nil {
			// the default handler just responds "hello, world"
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`hello, world`))
			})
		}

		// HTTP/3
		if l.QUIC {
			// create a udp listener on the specified port
			udpListener := runtimex.Try1(serverStack.ListenUDP("udp", &net.UDPAddr{
				IP:   net.ParseIP(s.ServerAddr),
				Port: l.Port,
				Zone: "",
			}))

			// create HTTP3 server using udpListener as underlying [net.PacketConn]
			http3Server := &http3.Server{
				TLSConfig: serverStack.ServerTLSConfig(),
				Handler:   handler,
			}

			// we need to track the UDP socket [net.PacketConn] and HTTP3 server to close them later
			// (closing the server does not close the connection)
			closables = append(closables, udpListener, http3Server)

			// start serving
			go http3Server.Serve(udpListener)
			continue
		}

		// HTTPS
		// create a tcp listener on the specified port
		tcpListener := runtimex.Try1(serverStack.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.ParseIP(s.ServerAddr),
			Port: l.Port,
			Zone: "",
		}))
		httpServer := &http.Server{
			TLSConfig: serverStack.ServerTLSConfig(),
			Handler:   handler,
		}
		closables = append(closables, httpServer)
		// start serving
		go httpServer.ServeTLS(tcpListener, "", "")
	}

	return closables
}

// NewEnvironment creates a new QA environment. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewEnvironment(clientConfig *ClientConfig, serversConfig *ServersConfig) *Environment {
	// create a new star topology
	topology := runtimex.Try1(netem.NewStarTopology(model.DiscardLogger))

	// create a DPIEngine for implementing censorship experiments can
	// plug in different types of DPIs, e.g. to drop all packets using a certain SNI
	dpi := netem.NewDPIEngine(model.DiscardLogger)

	// create array of closables that we track to close them later
	var closables []io.Closer

	// create a client and its DNS server (which we need to close later)
	clientStack, clientDNS := configureClient(clientConfig, topology, dpi)
	closables = append(closables, clientDNS)

	// set the default resolver address for the servers's DNS
	resolverAddr := serversConfig.ResolverAddr
	if resolverAddr == "" {
		resolverAddr = DefaultServersResolver
	}

	// create DNS server stack for the servers
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	//
	// note: we need to add a little bit of delay to the router<->servers
	// path such that rules that use spoofing always determinstically
	// succeed in spoofing the packets (w/o delays it's flaky).
	dnsServerStack := runtimex.Try1(topology.AddHost(
		resolverAddr, // server IP address
		resolverAddr, // default resolver address
		&netem.LinkConfig{
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))

	// create DNS server
	serversDNS := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		dnsServerStack,
		resolverAddr,
		serversConfig.DNSConfig,
	))
	closables = append(closables, serversDNS)

	// create and launch HTTP servers on the server stack
	// (and track them so we can close them at a later time)
	for _, s := range serversConfig.Servers {
		closables = append(closables, configureServer(&s, topology, resolverAddr)...)
	}

	return &Environment{
		clientStack: clientStack,
		dpi:         dpi,
		closables:   closables,
		topology:    topology,
	}
}

// DPIEngine returns the [netem.DPIEngine] we're using on the
// link between the client stack and the router. You can safely
// add new DPI rules from concurrent goroutines at any time.
func (e *Environment) DPIEngine() *netem.DPIEngine {
	return e.dpi
}

// Do executes the given function such that [netxlite] code uses the
// underlying clientStack rather than ordinary networking code.
func (e *Environment) Do(function func()) {
	WithCustomTProxy(e.clientStack, function)
}

// Close closes all the resources used by [Environment].
func (e *Environment) Close() error {
	for _, c := range e.closables {
		c.Close()
	}
	e.topology.Close()
	return nil
}
