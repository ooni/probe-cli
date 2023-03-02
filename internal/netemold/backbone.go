package netem

//
// Emulates a backbone
//

import (
	"context"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Backbone is a network backbone. The zero value is invalid; please,
// use [NewBackbone] to create a new valid instance.
//
// The backbone creates the following network topology:
//
//	   left                              right
//
//	.--------.           L1
//	| client | --------------------.
//	'--------'                      \
//	                                 \
//	                                  .----------.
//	                                  | backbone |
//	     .-------.                    '----------'
//	  .--------. | ------------------/
//	.--------. | -------------------/
//	| server | --------------------'
//	'--------'         L2..LN
//
// Where, L1, L2, ..., LN are [Link]s.
//
// Hence, going from client to backbone is going in the left->right direction of
// the L1 link. On the contrary, going from the server to the backbone is
// going in the right->left direction of the server-specific Lx for x in 2..N.
//
// The [Backbone] will remember the IP address of each configured client and
// server and will route traffic accordingly.
type Backbone struct {
	// mu provides mutual exclusion.
	mu sync.Mutex

	// table is the routing table.
	table map[string]*NIC
}

// NewBackbone creates a new backbone instance.
func NewBackbone() *Backbone {
	return &Backbone{
		mu:    sync.Mutex{},
		table: map[string]*NIC{},
	}
}

// AddClient adds a client stub network to the backbone. This function starts
// background goroutines that implement routing such that packets destined
// to the client IP address will reach the client. Those goroutines will run
// as long as the given context has not been canceled. This function will
// panic if the stack has an IP address belonging to a client or server that
// has previously been registered. The [Link] created for the client will
// use the configured [LinkDPIEngine] (use [DPINone] to disable DPI).
func (b *Backbone) AddClient(
	ctx context.Context,
	stack *GvisorStack,
	factory LinkFactory,
	dpi LinkDPIEngine,
) {
	defer b.mu.Unlock()
	b.mu.Lock()

	// make sure we don't have duplicate IP addresses
	_, found := b.table[stack.IPAddress()]
	runtimex.Assert(!found, "netem: Router: detected duplicate IP address")

	// create the client and the internet NIC
	localNIC := NewNIC()
	internetNIC := NewNIC()

	// connect the NICs using a link and install the DPI engine
	link := factory(localNIC, internetNIC, dpi)
	link.Up(ctx)

	// attach the stack to its NIC
	stack.Attach(ctx, localNIC)

	// route traffic exiting on the internetNIC
	go b.routeLoop(ctx, internetNIC)

	// register the internetNIC with network with the backbone
	b.table[stack.IPAddress()] = internetNIC
	log.Infof("netem: route add %s %s", stack.IPAddress(), internetNIC.Name)
}

// AddServer is like [AddClient] but adds a server to the backbone.
func (b *Backbone) AddServer(ctx context.Context, stack *GvisorStack, factory LinkFactory) {
	b.AddClient(ctx, stack, factory, &DPINone{})
}

// route routes traffic emitted by a given NIC to the correct destination NIC.
func (b *Backbone) routeLoop(ctx context.Context, nic *NIC) {
	for {
		rawPacket, err := nic.ReadIncoming(ctx)
		if err != nil {
			log.Warnf("netem: routeLoop: %s", err.Error())
			return
		}
		b.maybeRoutePacket(ctx, rawPacket)
	}
}

// maybeRoutePacket attempts to route a raw packet.
func (b *Backbone) maybeRoutePacket(ctx context.Context, rawInput []byte) {
	// parse the packet
	packet, err := dissect(rawInput)
	if err != nil {
		log.Warnf("netem: maybeRoutePacket: %s", err.Error())
		return
	}

	// decrement the TTL and drop the packet if TTL exceeded in transit
	if ttl := packet.timeToLive(); ttl <= 0 {
		log.Warn("netem: maybeRoutePacket: TTL exceeded in transit")
		return
	}
	packet.decrementTimeToLive()

	// figure out the interface where to emit the packet.
	destAddr := packet.destinationIPAddress()
	b.mu.Lock()
	destNIC := b.table[destAddr]
	b.mu.Unlock()
	if destNIC == nil {
		log.Warnf("netem: maybeRoutePacket: %s: no route to host", destAddr)
		return
	}

	// serialize a TCP or UDP packet while ignoring other protocols
	rawOutput, err := packet.serialize()
	if err != nil {
		log.Warnf("netem: maybeRoutePacket: %s", err.Error())
		return
	}

	// emit the packet on the destination interface
	destNIC.WriteOutgoing(ctx, rawOutput)
}
