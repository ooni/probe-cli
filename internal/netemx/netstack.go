package netemx

import "github.com/ooni/netem"

// NetStackServerFactory constructs a new [NetStackServer].
type NetStackServerFactory interface {
	// MustNewServer constructs a [NetStackServer] BORROWING a reference to an
	// underlying network attached to an userspace TCP/IP stack. This method MAY
	// call PANIC in case of failure.
	MustNewServer(stack *netem.UNetStack) NetStackServer
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
