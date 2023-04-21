package netemx

import (
	"net"
	"net/http"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// The netemx environment design is based on netemx_test.

// Environment is a configurable [netem] QA environment
// with a DNS server stack, multiple server stacks, and a client stack.
type Environment struct {
	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// dnsServer is the DNS server.
	dnsServer *netem.DNSServer

	// dpi refers to the [netem.DPIEngine] we're using.
	dpi *netem.DPIEngine

	// httpServers are the HTTP servers.
	httpServers []*http.Server

	// http3Servers are the HTTP/3 servers.
	http3Servers []*http3.Server

	// topology is the topology we're using.
	topology *netem.StarTopology
}

type Config struct {
	// ClientAddr is the OPTIONAL address of the client stack.
	// If empty, we use 10.0.0.14
	ClientAddr string
	// DNSConfig is the MANDATORY [*netem.DNSConfig] to be used for the DNS in this environment.
	DNSConfig *netem.DNSConfig
	// Resolver is the OPTIONAL address of the default resolver to be used in the environment.
	// If empty, we use 1.1.1.1
	Resolver string
	// Servers is the MANDATORY list of [ServerStack]s to be used in this environment.
	Servers []ServerStack
}

type ServerStack struct {
	// ServerAddr is the MANDATORY address of the web server stack.
	ServerAddr string
	// Listeners is the MANDATORY list of [Listener], i.e. server instances on this stack.
	Listeners []Listener
}

type Listener struct {
	// Port is the port that this listener is running on.
	Port int
	// QUIC indicates whether this listener uses QUIC instead of TCP as transport
	QUIC bool
}

// NewEnvironment creates a new QA environment. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewEnvironment(config Config) *Environment {
	// create a new star topology
	topology := runtimex.Try1(netem.NewStarTopology(model.DiscardLogger))

	// set the default resolver address
	resolverAddr := config.Resolver
	if resolverAddr == "" {
		resolverAddr = "1.1.1.1"
	}

	// create server stacks
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	dnsServerStack := runtimex.Try1(topology.AddHost(
		resolverAddr, // server IP address
		resolverAddr, // default resolver address
		&netem.LinkConfig{},
	))

	// create DNS server using the dnsServerStack
	dnsServer := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		dnsServerStack,
		resolverAddr,
		config.DNSConfig,
	))

	// create HTTP servers
	var servers []*http.Server
	var servers3 []*http3.Server
	for _, s := range config.Servers {
		serverStack := runtimex.Try1(topology.AddHost(
			s.ServerAddr, // server IP address
			resolverAddr, // default resolver address
			&netem.LinkConfig{},
		))

		for _, l := range s.Listeners {
			// create HTTP server using the server stack
			if l.QUIC {
				udpListener := runtimex.Try1(serverStack.ListenUDP("udp", &net.UDPAddr{
					IP:   net.ParseIP(s.ServerAddr),
					Port: l.Port,
					Zone: "",
				}))
				http3Server := &http3.Server{
					TLSConfig: serverStack.ServerTLSConfig(),
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Write([]byte(`hello, world`))
					}),
				}
				servers3 = append(servers3, http3Server)
				go http3Server.Serve(udpListener)
				continue
			}
			tlsListener := runtimex.Try1(serverStack.ListenTCP("tcp", &net.TCPAddr{
				IP:   net.ParseIP(s.ServerAddr),
				Port: l.Port,
				Zone: "",
			}))
			httpServer := &http.Server{
				TLSConfig: serverStack.ServerTLSConfig(),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`hello, world`))
				}),
			}
			servers = append(servers, httpServer)
			go httpServer.ServeTLS(tlsListener, "", "")
		}
	}

	// create a DPIEngine for implementing censorship
	dpi := netem.NewDPIEngine(model.DiscardLogger)

	// create client stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	clientStack := runtimex.Try1(topology.AddHost(
		"10.0.0.14",  // client IP address
		resolverAddr, // default resolver address
		&netem.LinkConfig{
			DPIEngine: dpi,
		},
	))

	return &Environment{
		clientStack:  clientStack,
		dnsServer:    dnsServer,
		dpi:          dpi,
		httpServers:  servers,
		http3Servers: servers3,
		topology:     topology,
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
	e.dnsServer.Close()
	for _, s := range e.httpServers {
		s.Close()
	}
	for _, s := range e.http3Servers {
		s.Close()
	}
	e.topology.Close()
	return nil
}
