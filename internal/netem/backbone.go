package netem

//
// Emulates a backbone
//

import (
	"errors"
	"net"
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
	// links is the list of links managed by the backbone.
	links []*Link

	// mu provides mutual exclusion.
	mu sync.Mutex

	// stacks is the list of stacks managed by the backbone.
	stacks []BackboneStack

	// table is the routing table.
	table map[string]writableBackboneNIC
}

// NewBackbone creates a new backbone instance.
func NewBackbone() *Backbone {
	return &Backbone{
		links: []*Link{},
		mu:    sync.Mutex{},
		table: map[string]writableBackboneNIC{},
	}
}

// Close closes the backbone
func (b *Backbone) Close() error {
	defer b.mu.Unlock()
	b.mu.Lock()

	// #FastFact: the order in which we're closing things here is
	// correct and swapping the order is less desirable

	for _, nic := range b.table {
		nic.Close()
	}

	for _, stack := range b.stacks {
		stack.Close()
	}

	for _, link := range b.links {
		link.Close()
	}

	return nil
}

// BackboneStack is the [Backbone] view of a [UNetStack].
type BackboneStack interface {
	// A BackboneStack is also a [LinkNIC]
	LinkNIC

	// IPAddress returns the stack IP address.
	IPAddress() string

	// Close closes the stack.
	Close() error
}

// AddStack adds a user-mode network stack to the backbone. This function starts
// background goroutines that implement routing such that packets destined
// to the client IP address will reach the client. Those goroutines will run
// as long as you call [Backbone.Close]. This function will panic if the stack
// uses an IP address that has already been registered. This function will
// additionally TAKE OWNERSHIP of the provided stack and Close it when
// the user calls the [Backbone.Close] method.
func (b *Backbone) AddStack(stack BackboneStack, linkConfig *LinkConfig) {
	defer b.mu.Unlock()
	b.mu.Lock()

	// make sure we don't have duplicate IP addresses
	_, found := b.table[stack.IPAddress()]
	runtimex.Assert(!found, "netem: Router: detected duplicate IP address")

	// create a [BackboneNIC] for this stack
	bn := newBackboneNIC(stack.InterfaceName(), stack.IPAddress())

	// connect the stack and the backbone using a link
	link := NewLink(stack, bn, linkConfig)

	// route traffic exiting on the backbone NIC
	go b.routeLoop(bn)

	// register everything with the backbone
	b.links = append(b.links, link)
	b.stacks = append(b.stacks, stack)
	b.table[stack.IPAddress()] = bn
}

// route routes traffic emitted by a given NIC to the correct destination NIC.
func (b *Backbone) routeLoop(nic readableBackboneNIC) {
	for {
		rawPacket, err := nic.readIncomingPacket()
		if err != nil {
			log.Warnf("netem: routeLoop: %s", err.Error())
			return
		}
		b.maybeRoutePacket(rawPacket)
	}
}

// maybeRoutePacket attempts to route a raw packet.
func (b *Backbone) maybeRoutePacket(rawInput []byte) {
	// parse the packet
	packet, err := dissectPacket(rawInput)
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
	destNIC.writeOutgoingPacket(rawOutput)
}

// readableBackboneNIC is the readable-from-the-backbone
// view of a [BackboneNIC].
type readableBackboneNIC interface {
	readIncomingPacket() ([]byte, error)
}

// writableBackboneNIC is the writable-from-the-backbone
// view of a [BackboneNIC].
type writableBackboneNIC interface {
	writeOutgoingPacket(packet []byte) error
	Close() error
}

// backboneNIC is a NIC attached to the backbone. The zero
// value is invalid; use [newBackboneNIC].
type backboneNIC struct {
	// closeOnce provides "once" semantics to close.
	closeOnce sync.Once

	// closed is closed when the backbone NIC is closed.
	closed chan any

	// ipAddress is the IP address we're using.
	ipAddress string

	// incoming collects packets from a stacks to the backbone.
	incoming chan []byte

	// name is the interface name
	name string

	// outgoing collects packets from the backbone to a stack.
	outgoing chan []byte

	// remoteIf is the remote interface
	remoteIf string
}

var _ LinkNIC = &backboneNIC{}

// newBackboneNIC creates a new [backboneNIC].
func newBackboneNIC(remoteIf, ipAddress string) *backboneNIC {
	name := remoteIf + ":1"
	log.Infof("netem: ifconfig %s up", name)
	log.Infof("netem: route add %s/32 %s", ipAddress, remoteIf)
	log.Infof("netem: route add default %s", name)
	return &backboneNIC{
		closeOnce: sync.Once{},
		closed:    make(chan any),
		ipAddress: ipAddress,
		incoming:  make(chan []byte, 1024),
		name:      name,
		outgoing:  make(chan []byte, 1024),
		remoteIf:  remoteIf,
	}
}

// InterfaceName implements LinkNIC
func (bn *backboneNIC) InterfaceName() string {
	return bn.name
}

// readIncomingPacket is called by the [Backbone]
func (bn *backboneNIC) readIncomingPacket() ([]byte, error) {
	select {
	case <-bn.closed:
		return nil, net.ErrClosed
	case packet := <-bn.incoming:
		return packet, nil
	}
}

// ReadPacket implements LinkNIC. This function is called by
// the [Link] and reads outoing packets.
func (bn *backboneNIC) ReadPacket() ([]byte, error) {
	select {
	case <-bn.closed:
		return nil, net.ErrClosed
	case packet := <-bn.outgoing:
		return packet, nil
	}
}

// ErrDropped indicates that a packet was dropped.
var ErrDropped = errors.New("netem: packet was dropped")

// writeOutgoingPacket is called by the [Backbone]
func (bn *backboneNIC) writeOutgoingPacket(packet []byte) error {
	select {
	case <-bn.closed:
		return net.ErrClosed
	case bn.outgoing <- packet:
		return nil
	default:
		return ErrDropped
	}
}

// WritePacket implements LinkNI. This function is called by
// the [Link] and writes incoming packets.
func (bn *backboneNIC) WritePacket(packet []byte) error {
	select {
	case <-bn.closed:
		return net.ErrClosed
	case bn.incoming <- packet:
		return nil
	default:
		return ErrDropped
	}
}

// Close closes the NIC
func (bn *backboneNIC) Close() error {
	bn.closeOnce.Do(func() {
		log.Infof("netem: route del %s/32 %s", bn.ipAddress, bn.remoteIf)
		log.Infof("netem: route del default %s", bn.name)
		log.Infof("netem: ifconfig %s down", bn.name)
		close(bn.closed)
	})
	return nil
}
