package netemx

//
// QA environment
//

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// qaEnvConfig is the private configuration for [MustNewQAEnv].
type qaEnvConfig struct {
	// clientAddress is the client IP address to use.
	clientAddress string

	// clientNICWrapper is the OPTIONAL wrapper for the client NIC.
	clientNICWrapper netem.LinkNICWrapper

	// ispResolver is the ISP resolver to use.
	ispResolver string

	// logger is the logger to use.
	logger model.Logger

	// netStacks contains information about the net stacks to create.
	netStacks map[string][]NetStackServerFactory

	// rootResolver is the root resolver address to use.
	rootResolver string
}

// QAEnvOption is an option to modify [NewQAEnv] default behavior.
type QAEnvOption func(config *qaEnvConfig)

// QAEnvOptionClientAddress sets the client IP address. If you do not set this option
// we will use [DefaultClientAddress].
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

// QAEnvOptionHTTPServer adds the given HTTP handler factory. If you do
// not set this option we will not create any HTTP server. Note that this
// option is just syntactic sugar for calling [QAEnvOptionNetStack]
// with the following three factories as argument:
//
// - [HTTPCleartextServerFactory] with port 80/tcp;
//
// - [HTTPSecureServerFactory] with port 443/tcp and nil TLSConfig;
//
// - [HTTP3ServerFactory] with port 443/udp and nil TLSConfig.
//
// We wrote this syntactic sugar factory because it covers the common case
// where you want support for HTTP, HTTPS, and HTTP3.
func QAEnvOptionHTTPServer(ipAddr string, factory HTTPHandlerFactory) QAEnvOption {
	runtimex.Assert(net.ParseIP(ipAddr) != nil, "not an IP addr")
	runtimex.Assert(factory != nil, "passed a nil handler factory")
	return qaEnvOptionNetStack(ipAddr, &HTTPCleartextServerFactory{
		Factory: factory,
		Ports:   []int{80},
	}, &HTTPSecureServerFactory{
		Factory:   factory,
		Ports:     []int{443},
		TLSConfig: nil, // use netem's default
	}, &HTTP3ServerFactory{
		Factory:   factory,
		Ports:     []int{443},
		TLSConfig: nil, // use netem's default
	})
}

// QAEnvOptionLogger sets the logger to use. If you do not set this option we
// will use [model.DiscardLogger] as the logger.
func QAEnvOptionLogger(logger model.Logger) QAEnvOption {
	return func(config *qaEnvConfig) {
		config.logger = logger
	}
}

// QAEnvOptionNetStack creates an userspace network stack with the given IP address and binds it
// to the given factory, which will be responsible to create listening sockets and closing them
// when we're done running. Examples of factories you can use with this method are:
//
// - [NewTCPEchoServerFactory];
//
// - [HTTPCleartextServerFactory];
//
// - [HTTPSecureServerFactory];
//
// - [HTTP3ServerFactory];
//
// - [UDPResolverFactory].
//
// Calling this method multiple times is equivalent to calling this method once with several
// factories. This would work as long as you do not specify the same port multiple times, otherwise
// the second bind attempt for an already bound port would fail.
//
// This function PANICS if you try to configure [ISPResolverAddress] or [RootResolverAddress]
// because these two addresses are already configured by [MustNewQAEnv].
func QAEnvOptionNetStack(ipAddr string, factories ...NetStackServerFactory) QAEnvOption {
	runtimex.Assert(
		ipAddr != ISPResolverAddress && ipAddr != RootResolverAddress,
		"QAEnvOptionNetStack: cannot configure RootResolverAddress or ISPResolverAddress",
	)
	return qaEnvOptionNetStack(ipAddr, factories...)
}

func qaEnvOptionNetStack(ipAddr string, factories ...NetStackServerFactory) QAEnvOption {
	return func(config *qaEnvConfig) {
		config.netStacks[ipAddr] = append(config.netStacks[ipAddr], factories...)
	}
}

// QAEnv is the environment for running QA tests using [github.com/ooni/netem]. The zero
// value of this struct is invalid; please, use [NewQAEnv].
type QAEnv struct {
	// baseLogger is the base [model.Logger] to use.
	baseLogger model.Logger

	// clientNICWrapper is the OPTIONAL wrapper for the client NIC.
	clientNICWrapper netem.LinkNICWrapper

	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// closables contains all entities where we have to take care of closing.
	closables []io.Closer

	// emulateAndroidGetaddrinfo controls whether to emulate the behavior of our wrapper for
	// the android implementation of getaddrinfo returning android_dns_cache_no_data
	emulateAndroidGetaddrinfo *atomic.Bool

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

// MustNewQAEnv creates a new [QAEnv]. This function PANICs on failure.
func MustNewQAEnv(options ...QAEnvOption) *QAEnv {
	// initialize the configuration
	config := &qaEnvConfig{
		clientAddress:    DefaultClientAddress,
		clientNICWrapper: nil,
		ispResolver:      ISPResolverAddress,
		logger:           model.DiscardLogger,
		rootResolver:     RootResolverAddress,
		netStacks:        map[string][]NetStackServerFactory{},
	}
	for _, option := range options {
		option(config)
	}

	// make sure we're going to create the ISP's DNS resolver.
	qaEnvOptionNetStack(config.ispResolver, &dnsOverUDPServerFactoryForGetaddrinfo{})(config)

	// make sure we're going to create the root DNS resolver.
	qaEnvOptionNetStack(config.rootResolver, &DNSOverUDPServerFactory{})(config)

	// use a prefix logger for the QA env
	prefixLogger := &logx.PrefixLogger{
		Prefix: fmt.Sprintf("%-16s", "NETEM"),
		Logger: config.logger,
	}

	// create an empty QAEnv
	env := &QAEnv{
		baseLogger:                config.logger,
		clientNICWrapper:          config.clientNICWrapper,
		clientStack:               nil,
		closables:                 []io.Closer{},
		emulateAndroidGetaddrinfo: &atomic.Bool{},
		ispResolverConfig:         netem.NewDNSConfig(),
		dpi:                       netem.NewDPIEngine(prefixLogger),
		once:                      sync.Once{},
		otherResolversConfig:      netem.NewDNSConfig(),
		topology:                  runtimex.Try1(netem.NewStarTopology(prefixLogger)),
	}

	// create all the required internals
	env.clientStack = env.mustNewClientStack(config)
	env.closables = append(env.closables, env.mustNewNetStacks(config)...)

	return env
}

func (env *QAEnv) mustNewClientStack(config *qaEnvConfig) *netem.UNetStack {
	// Note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	//
	// TODO(bassosimone,kelmenhorst): consider allowing to configure the
	// delays and losses should the need for this arise in the future.
	return runtimex.Try1(env.topology.AddHost(
		DefaultClientAddress,
		config.ispResolver,
		&netem.LinkConfig{
			DPIEngine:        env.dpi,
			LeftNICWrapper:   env.clientNICWrapper,
			LeftToRightDelay: time.Millisecond,
			RightToLeftDelay: time.Millisecond,
		},
	))
}

func (env *QAEnv) mustNewNetStacks(config *qaEnvConfig) (closables []io.Closer) {
	resolver := config.rootResolver

	for ipAddr, factories := range config.netStacks {
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

		for _, factory := range factories {
			// instantiate a server with the given underlying network
			server := factory.MustNewServer(env, stack)

			// listen and start serving in the background
			server.MustStart()

			// track the server as the something that needs to be closed
			closables = append(closables, server)
		}
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

// Logger is the [model.Logger] configured for this [*QAEnv],
func (env *QAEnv) Logger() model.Logger {
	return env.baseLogger
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

// EmulateAndroidGetaddrinfo configures [QAEnv] such that the Do method wraps
// the underlying client stack to return android_dns_cache_no_data on any error
// that occurs. This method can be safely called by multiple goroutines.
func (env *QAEnv) EmulateAndroidGetaddrinfo(value bool) {
	env.emulateAndroidGetaddrinfo.Store(value)
}

// Do executes the given function such that [netxlite] code uses the
// underlying clientStack rather than ordinary networking code.
func (env *QAEnv) Do(function func()) {
	var stack netem.UnderlyingNetwork = env.clientStack
	if env.emulateAndroidGetaddrinfo.Load() {
		stack = &androidStack{stack}
	}
	WithCustomTProxy(stack, function)
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
