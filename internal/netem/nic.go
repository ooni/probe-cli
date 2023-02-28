package netem

//
// Network interface controller (NIC) emulation
//

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// NIC is a network interface controller. The zero value is
// invalid; you MUST use [NewNIC] to create a [NIC].
//
// Once you have a [NIC] instance you can:
//
//   - attach the [NIC] to a [GvisorStack] such that the
//     stack ends up using the [NIC];
//
//   - use a [Link] to collect two [NIC]s.
//
// Internally a [NIC] uses channels to represent incoming and
// outgoing IPv4 or IPv6 packets. We deal with raw IPv4 and IPv6
// packets because [GvisorStack] reads and writes this kind of
// data through its internal, userspace TUN interface.
//
// Reading either queue blocks until either a new packet arrives
// or the controlling context has been canceled. Writing a new
// packet does not block. We create channels with queues and when
// the queue is fully, we discaring extra packets.
type NIC struct {
	// incoming queues incoming packets.
	incoming chan []byte

	// name is the NIC name.
	name string

	// outgoing queue outgoing packets.
	outgoing chan []byte
}

// NICOption is an option for [NewNic].
type NICOption func(nic *NIC)

// nicIndex is the index used to name NICs.
var nicIndex = &atomic.Int64{}

// DefaultNICBufferSize is the default channel buffer size used by [NewNIC].
const DefaultNICBufferSize = 1024

// NICOptionIncomingBufferSize selects the number of full-size packets
// that the NICs incoming buffer should hold before dropping packets. The
// default is to use a [DefaultNICBuffersize]-entries buffer.
func NICOptionIncomingBufferSize(value int) NICOption {
	return func(nic *NIC) {
		nic.incoming = make(chan []byte, value)
	}
}

// NICOptionOutgoingBufferSize selects the number of full-size packets
// that the NICs outgoing buffer should hold before dropping packets. The
// default is to use a [DefaultNICBuffersize]-entries buffer.
func NICOptionOutgoingBufferSize(value int) NICOption {
	return func(nic *NIC) {
		nic.outgoing = make(chan []byte, value)
	}
}

// NICOptionName selects the name of the NIC. The default is to use "ethX"
// where X is a global, atomic integer we increment for each new NIC.
func NICOptionName(value string) NICOption {
	return func(nic *NIC) {
		nic.name = value
	}
}

// NewNIC creates a new NIC instance using the given options.
func NewNIC(options ...NICOption) *NIC {
	nic := &NIC{
		incoming: make(chan []byte, DefaultNICBufferSize),
		name:     fmt.Sprintf("eth%d", nicIndex.Add(1)),
		outgoing: make(chan []byte, DefaultNICBufferSize),
	}
	for _, opt := range options {
		opt(nic)
	}
	return nic
}

// ReadIncoming reads a raw packet from the incoming channel or
// returns an error if the given context is done.
func (n *NIC) ReadIncoming(ctx context.Context) ([]byte, error) {
	select {
	case rawPacket := <-n.incoming:
		return rawPacket, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReadOutgoing reads a raw packet from the outgoing channel or
// returns an error if the given context is done.
func (n *NIC) ReadOutgoing(ctx context.Context) ([]byte, error) {
	select {
	case rawPacket := <-n.outgoing:
		return rawPacket, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ErrNICBufferFull indicates that a NIC's buffer is full.
var ErrNICBufferFull = errors.New("nic: buffer is full: dropping packet")

// WriteIncoming writes a raw packet from the incoming channel or
// returns an error if the context is done or the buffer full.
func (n *NIC) WriteIncoming(ctx context.Context, rawPacket []byte) error {
	select {
	case n.incoming <- rawPacket:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrNICBufferFull
	}
}

// WriteOutgoing writes a raw packet from the outgoing channel or
// returns an error if the context is done or the buffer full.
func (n *NIC) WriteOutgoing(ctx context.Context, rawPacket []byte) error {
	select {
	case n.outgoing <- rawPacket:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrNICBufferFull
	}
}
