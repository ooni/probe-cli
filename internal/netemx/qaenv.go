package netemx

//
// QA environment
//

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go/http3"
)

// QAEnvDefaultClientAddress is the default client IP address.
const QAEnvDefaultClientAddress = "10.0.0.17"

// QAEnvDefaultISPResolverAddress is the default IP address of the client ISP resolver.
const QAEnvDefaultISPResolverAddress = "10.0.0.34"

// QAEnvDefaultUncensoredResolverAddress is the default uncensored resolver IP address.
const QAEnvDefaultUncensoredResolverAddress = "1.1.1.1"

type qaEnvConfig struct {
	// clientAddress is the client IP address to use.
	clientAddress string

	// clientNICWrapper is the OPTIONAL wrapper for the client NIC.
	clientNICWrapper netem.LinkNICWrapper

	// dnsOverUDPResolvers contains the DNS-over-UDP resolvers to create.
	dnsOverUDPResolvers []string

	// httpServers contains the HTTP servers to create.
	httpServers map[string]http.Handler

	// ispResolver is the ISP resolver to use.
	ispResolver string

	// logger is the logger to use.
	logger model.Logger

	// netStacks contains information about the net stacks to create.
	netStacks map[string]QAEnvNetStackHandler
}

// QAEnvOption is an option to modify [NewQAEnv] default behavior.
type QAEnvOption func(config *qaEnvConfig)

// QAEnvOptionClientAddress sets the client IP address. If you do not set this option
// we will use [QAEnvDefaultClientAddress].
func QAEnvOptionClientAddress(ipAddr string) QAEnvOption {
	runtimex.Assert(net.ParseIP(ipAddr) != nil, "not an IP addr")
	return func(config *qaEnvConfig) {
		config.clientAddress = ipAddr
	}
}

// QAEnvOptionClientNICWrapper sets the NIC wrapper for the client. The most common use case
// for this functionality is capturing packets using [netem.NewPCAPDumper].
func QAEnvOptionClientNICWrapper(wrapper netem.LinkNICWrapper) QAEnvOption {
	return func(config *qaEnvConfig) {
		config.clientNICWrapper = wrapper
	}
}

// QAEnvOptionDNSOverUDPResolvers adds the given DNS-over-UDP resolvers. If you do not set this option
// we will create a single resolver using [QAEnvDefaultUncensoredResolverAddress].
func QAEnvOptionDNSOverUDPResolvers(ipAddrs ...string) QAEnvOption {
	for _, a := range ipAddrs {
		runtimex.Assert(net.ParseIP(a) != nil, "not an IP addr")
	}
	return func(config *qaEnvConfig) {
		config.dnsOverUDPResolvers = append(config.dnsOverUDPResolvers, ipAddrs...)
	}
}

// QAEnvOptionHTTPServer adds the given HTTP server. If you do not set this option
// we will not create any HTTP server.
func QAEnvOptionHTTPServer(ipAddr string, handler http.Handler) QAEnvOption {
	runtimex.Assert(net.ParseIP(ipAddr) != nil, "not an IP addr")
	runtimex.Assert(handler != nil, "passed a nil handler")
	return func(config *qaEnvConfig) {
		config.httpServers[ipAddr] = handler
	}
}

// QAEnvOptionISPResolverAddress sets the ISP's resolver IP address. If you do not set this option
// we will use [QAEnvDefaultISPResolverAddress] as the address.
func QAEnvOptionISPResolverAddress(ipAddr string) QAEnvOption {
	runtimex.Assert(net.ParseIP(ipAddr) != nil, "not an IP addr")
	return func(config *qaEnvConfig) {
		config.ispResolver = ipAddr
	}
}

// QAEnvOptionLogger sets the logger to use. If you do not set this option we
// will use [model.DiscardLogger] as the logger.
func QAEnvOptionLogger(logger model.Logger) QAEnvOption {
	return func(config *qaEnvConfig) {
		config.logger = logger
	}
}

// QAEnvNetStackHandler handles a [*netem.UNetStack] created using [QAEnvOptionNetStack].
type QAEnvNetStackHandler interface {
	// Listen should use the stack to create all the listening TCP and UDP sockets
	// required by the specific test case, as well as to start the required background
	// goroutines servicing incoming requests for the created listeners. This method
	// MUST BE CONCURRENCY SAFE and it MUST NOT arrange for the Close method to close
	// the stack because it is managed by the [QAEnv]. This method MAY call PANIC
	// in case of listening failure: the caller calls PANIC on error anyway.
	Listen(stack *netem.UNetStack) error

	// Close should close the listening TCP and UDP sockets and the background
	// goroutines created by Listen. This method MUST BE CONCURRENCY SAFE and IDEMPOTENT and
	// it MUST NOT close the stack passed to Listen because it is managed by [QAEnv].
	Close() error
}

// QAEnvOptionNetStack creates an userspace network stack with the given IP address and binds it
// to the given handler, which will be responsible to create listening sockets and closing them
// when we're done running. This option is lower-level than [QAEnvOptionHTTPServer], so you should
// probably use [QAEnvOptionHTTPServer] unless you need to do something custom.
func QAEnvOptionNetStack(ipAddr string, handler QAEnvNetStackHandler) QAEnvOption {
	return func(config *qaEnvConfig) {
		config.netStacks[ipAddr] = handler
	}
}

// QAEnv is the environment for running QA tests using [github.com/ooni/netem]. The zero
// value of this struct is invalid; please, use [NewQAEnv].
type QAEnv struct {
	// clientNICWrapper is the OPTIONAL wrapper for the client NIC.
	clientNICWrapper netem.LinkNICWrapper

	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// closables contains all entities where we have to take care of closing.
	closables []io.Closer

	// ispResolverConfig is the DNS config used by the ISP resolver.
	ispResolverConfig *netem.DNSConfig

	// dpi refers to the [netem.DPIEngine] we're using.
	dpi *netem.DPIEngine

	// once ensures Close has "once" semantics.
	once sync.Once

	// otherResolversConfig is the DNS config used by non-ISP resolvers.
	otherResolversConfig *netem.DNSConfig

	// topology is the topology we're using.
	topology *netem.StarTopology
}

// NewQAEnv creates a new [QAEnv].
func NewQAEnv(options ...QAEnvOption) *QAEnv {
	// initialize the configuration
	config := &qaEnvConfig{
		clientAddress:       QAEnvDefaultClientAddress,
		clientNICWrapper:    nil,
		dnsOverUDPResolvers: []string{},
		httpServers:         map[string]http.Handler{},
		ispResolver:         QAEnvDefaultISPResolverAddress,
		logger:              model.DiscardLogger,
		netStacks:           map[string]QAEnvNetStackHandler{},
	}
	for _, option := range options {
		option(config)
	}
	if len(config.dnsOverUDPResolvers) < 1 {
		config.dnsOverUDPResolvers = append(config.dnsOverUDPResolvers, QAEnvDefaultUncensoredResolverAddress)
	}

	// create an empty QAEnv
	env := &QAEnv{
		clientNICWrapper:     config.clientNICWrapper,
		clientStack:          nil,
		closables:            []io.Closer{},
		ispResolverConfig:    netem.NewDNSConfig(),
		dpi:                  netem.NewDPIEngine(config.logger),
		once:                 sync.Once{},
		otherResolversConfig: netem.NewDNSConfig(),
		topology:             runtimex.Try1(netem.NewStarTopology(config.logger)),
	}

	// create all the required internals
	env.closables = append(env.closables, env.mustNewISPResolverStack(config))
	env.clientStack = env.mustNewClientStack(config)
	env.closables = append(env.closables, env.mustNewResolvers(config)...)
	env.closables = append(env.closables, env.mustNewHTTPServers(config)...)
	env.closables = append(env.closables, env.mustNewNetStacks(config)...)

	return env
}

func (env *QAEnv) mustNewISPResolverStack(config *qaEnvConfig) io.Closer {
	// Create the ISP's DNS server TCP/IP stack.
	//
	// Note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	stack := runtimex.Try1(env.topology.AddHost(
		config.ispResolver, // server IP address
		config.ispResolver, // default resolver address
		&netem.LinkConfig{
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))

	// Create the client's DNS server using the stack.
	server := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		stack,
		config.ispResolver,
		env.ispResolverConfig,
	))

	return server
}

func (env *QAEnv) mustNewClientStack(config *qaEnvConfig) *netem.UNetStack {
	// Note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	//
	// TODO(bassosimone,kelmenhorst): consider allowing to configure the
	// delays and losses should the need for this arise in the future.
	return runtimex.Try1(env.topology.AddHost(
		QAEnvDefaultClientAddress,
		config.ispResolver,
		&netem.LinkConfig{
			DPIEngine:        env.dpi,
			LeftNICWrapper:   env.clientNICWrapper,
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))
}

func (env *QAEnv) mustNewResolvers(config *qaEnvConfig) (closables []io.Closer) {
	for _, addr := range config.dnsOverUDPResolvers {
		// Create the server's TCP/IP stack
		//
		// Note: because the stack is created using topology.AddHost, we don't
		// need to call Close when done using it, since the topology will do that
		// for us when we call the topology's Close method.
		stack := runtimex.Try1(env.topology.AddHost(
			addr, // IP address
			addr, // default resolver address
			&netem.LinkConfig{
				LeftToRightDelay: time.Millisecond,
				RightToLeftDelay: time.Millisecond,
			},
		))

		// create DNS server
		server := runtimex.Try1(netem.NewDNSServer(
			model.DiscardLogger,
			stack,
			addr,
			env.otherResolversConfig,
		))

		// track this closable
		closables = append(closables, server)
	}
	return
}

func (env *QAEnv) mustNewHTTPServers(config *qaEnvConfig) (closables []io.Closer) {
	runtimex.Assert(len(config.dnsOverUDPResolvers) >= 1, "expected at least one DNS resolver")
	resolver := config.dnsOverUDPResolvers[0]

	for addr, handler := range config.httpServers {
		// Create the server's TCP/IP stack
		//
		// Note: because the stack is created using topology.AddHost, we don't
		// need to call Close when done using it, since the topology will do that
		// for us when we call the topology's Close method.
		stack := runtimex.Try1(env.topology.AddHost(
			addr,     // IP address
			resolver, // default resolver address
			&netem.LinkConfig{
				LeftToRightDelay: time.Millisecond,
				RightToLeftDelay: time.Millisecond,
			},
		))

		// create HTTP, HTTPS and HTTP/3 servers for this stack
		closables = append(closables, env.mustCreateAllHTTPServers(stack, handler, addr)...)
	}
	return
}

func (env *QAEnv) mustCreateAllHTTPServers(
	stack *netem.UNetStack, handler http.Handler, addr string) (closables []io.Closer) {
	ipAddr := net.ParseIP(addr)
	runtimex.Assert(ipAddr != nil, "invalid IP addr")

	// listen for HTTP
	{
		listener := runtimex.Try1(stack.ListenTCP("tcp", &net.TCPAddr{IP: ipAddr, Port: 80}))
		srv := &http.Server{Handler: handler}
		closables = append(closables, srv)
		go srv.Serve(listener)
	}

	// listen for HTTPS
	{
		listener := runtimex.Try1(stack.ListenTCP("tcp", &net.TCPAddr{IP: ipAddr, Port: 443}))
		srv := &http.Server{TLSConfig: stack.ServerTLSConfig(), Handler: handler}
		closables = append(closables, srv)
		go srv.ServeTLS(listener, "", "")
	}

	// listen for HTTP3
	{
		listener := runtimex.Try1(stack.ListenUDP("udp", &net.UDPAddr{IP: ipAddr, Port: 443}))
		srv := &http3.Server{TLSConfig: stack.ServerTLSConfig(), Handler: handler}
		closables = append(closables, listener, srv)
		go srv.Serve(listener)

	}
	return
}

func (env *QAEnv) mustNewNetStacks(config *qaEnvConfig) (closables []io.Closer) {
	runtimex.Assert(len(config.dnsOverUDPResolvers) >= 1, "expected at least one DNS resolver")
	resolver := config.dnsOverUDPResolvers[0]

	for ipAddr, handler := range config.netStacks {
		// Create the server's TCP/IP stack
		//
		// Note: because the stack is created using topology.AddHost, we don't
		// need to call Close when done using it, since the topology will do that
		// for us when we call the topology's Close method.
		stack := runtimex.Try1(env.topology.AddHost(
			ipAddr,   // IP address
			resolver, // default resolver address
			&netem.LinkConfig{
				LeftToRightDelay: time.Millisecond,
				RightToLeftDelay: time.Millisecond,
			},
		))

		// create the required listeners
		runtimex.Try0(handler.Listen(stack))

		// track the handler as the something that needs to be closed
		closables = append(closables, handler)
	}
	return
}

// AddRecordToAllResolvers adds the given DNS record to all DNS resolvers. You can safely
// add new DNS records from concurrent goroutines at any time.
func (env *QAEnv) AddRecordToAllResolvers(domain string, cname string, addrs ...string) {
	env.ISPResolverConfig().AddRecord(domain, cname, addrs...)
	env.OtherResolversConfig().AddRecord(domain, cname, addrs...)
}

// ISPResolverConfig returns the [*netem.DNSConfig] of the ISP resolver. Note that can safely
// add new DNS records from concurrent goroutines at any time.
func (env *QAEnv) ISPResolverConfig() *netem.DNSConfig {
	return env.ispResolverConfig
}

// OtherResolversConfig returns the [*netem.DNSConfig] of the non-ISP resolvers. Note that can safely
// add new DNS records from concurrent goroutines at any time.
func (env *QAEnv) OtherResolversConfig() *netem.DNSConfig {
	return env.otherResolversConfig
}

// DPIEngine returns the [*netem.DPIEngine] we're using on the
// link between the client stack and the router. You can safely
// add new DPI rules from concurrent goroutines at any time.
func (env *QAEnv) DPIEngine() *netem.DPIEngine {
	return env.dpi
}

// Do executes the given function such that [netxlite] code uses the
// underlying clientStack rather than ordinary networking code.
func (env *QAEnv) Do(function func()) {
	WithCustomTProxy(env.clientStack, function)
}

// Close closes all the resources used by [QAEnv].
func (env *QAEnv) Close() error {
	env.once.Do(func() {
		// first close all the possible closables we track
		for _, c := range env.closables {
			c.Close()
		}

		// finally close the whole network topology
		env.topology.Close()
	})
	return nil
}

// QAEnvDefaultWebPage is the webpage returned by [QAEnvDefaultHTTPHandler].
// created for [ConfigHTTPServer].
const QAEnvDefaultWebPage = `<!doctype html>
<html>
<head>
    <title>Default Web Page</title>
</head>
<body>
<div>
    <h1>Default Web Page</h1>
    <p>This is the default web page of the default domain.</p>
</div>
</body>
</html>
`

// QAEnvDefaultHTTPHandler returns the default HTTP handler.
func QAEnvDefaultHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(QAEnvDefaultWebPage))
	})
}

// QAEnvNetStackTCPEcho is a [QAEnvNetStackHandler] implementing a TCP echo service.
func QAEnvNetStackTCPEcho(logger model.Logger, ports ...uint16) QAEnvNetStackHandler {
	return &qaEnvNetStackTCPEcho{
		closers: []io.Closer{},
		logger:  logger,
		mu:      sync.Mutex{},
		ports:   ports,
	}
}

type qaEnvNetStackTCPEcho struct {
	closers []io.Closer
	logger  model.Logger
	mu      sync.Mutex
	ports   []uint16
}

// Close implements QAEnvNetStackHandler.
func (echo *qaEnvNetStackTCPEcho) Close() error {
	// "this method MUST be CONCURRENCY SAFE"
	defer echo.mu.Unlock()
	echo.mu.Lock()

	// make sure we close all the child listeners
	for _, closer := range echo.closers {
		_ = closer.Close()
	}

	// "this method MUST be IDEMPOTENT"
	echo.closers = []io.Closer{}

	return nil
}

// Listen implements QAEnvNetStackHandler.
func (echo *qaEnvNetStackTCPEcho) Listen(stack *netem.UNetStack) error {
	// "this method MUST be CONCURRENCY SAFE"
	defer echo.mu.Unlock()
	echo.mu.Lock()

	// for each port of interest - note that here we panic liberally because we are
	// allowed to do so by the [QAEnvNetStackHandler] documentation.
	for _, port := range echo.ports {
		// create the endpoint address
		ipAddr := net.ParseIP(stack.IPAddress())
		runtimex.Assert(ipAddr != nil, "invalid IP address")
		epnt := &net.TCPAddr{IP: ipAddr, Port: int(port)}

		// attempt to listen
		listener := runtimex.Try1(stack.ListenTCP("tcp", epnt))

		// spawn goroutine for accepting
		go echo.acceptLoop(listener)

		// track this listener as something to close later
		echo.closers = append(echo.closers, listener)
	}
	return nil
}

func (echo *qaEnvNetStackTCPEcho) acceptLoop(listener net.Listener) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(echo.logger, "qaEnvNetStackTCPEcho.acceptLoop")
	for {
		conn := runtimex.Try1(listener.Accept())
		go echo.serve(conn)
	}
}

func (echo *qaEnvNetStackTCPEcho) serve(conn net.Conn) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(echo.logger, "qaEnvTCPListenerEcho.serve")

	// make sure we close the conn
	defer conn.Close()

	// loop until there is an I/O error
	for {
		buffer := make([]byte, 4096)
		count := runtimex.Try1(conn.Read(buffer))
		_, _ = conn.Write(buffer[:count])
	}
}
