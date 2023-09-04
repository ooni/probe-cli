package netemx

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NetStackServerFactoryEnv is [NetStackServerFactory] view of [*QAEnv].
type NetStackServerFactoryEnv interface {
	// Logger returns the base logger configured for the [*QAEnv].
	Logger() model.Logger

	// OtherResolversConfig returns the configuration used by all the
	// DNS resolvers except the ISP's DNS resolver.
	OtherResolversConfig() *netem.DNSConfig
}

// NetStackServerFactory constructs a new [NetStackServer].
type NetStackServerFactory interface {
	// MustNewServer constructs a [NetStackServer] BORROWING a reference to an
	// underlying network attached to an userspace TCP/IP stack. This method MAY
	// call PANIC in case of failure.
	MustNewServer(env NetStackServerFactoryEnv, stack *netem.UNetStack) NetStackServer
}

// NetStackServer handles the lifecycle of a server using a TCP/IP stack in userspace.
type NetStackServer interface {
	// MustStart uses the underlying stack to create all the listening TCP and UDP sockets
	// required by the specific test case, as well as to start the required background
	// goroutines servicing incoming requests for the created listeners. This method
	// MUST BE CONCURRENCY SAFE and it MUST NOT arrange for the Close method to close
	// the stack because it is managed by the [QAEnv]. This method MUST call PANIC
	// in case there is any error in listening or starting the required servers.
	MustStart()

	// Close should close the listening TCP and UDP sockets and the background
	// goroutines created by Listen. This method MUST BE CONCURRENCY SAFE and IDEMPOTENT.
	Close() error
}
