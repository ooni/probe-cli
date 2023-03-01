package netem

//
// Slow circuits for throttling
//

import (
	"context"
	"sync/atomic"
	"time"
)

// slowCircuitPacket is a packet sent over the slow circuit.
type slowCircuitPacket struct {
	// nic is the nic where to send the packet
	nic *NIC

	// rawPacket is the raw packet to send
	rawPacket []byte
}

// slowCircuit implements throttling by allowing DPI to
// route traffic over a slow circuit. The zero value
// is invalid; please, use [newSlowCircuit] to construct.
type slowCircuit struct {
	// adaptive controls whether we increase throttling.
	adaptive bool

	// ch is the channel used by the slow circuit.
	ch chan *slowCircuitPacket

	// n is the number of bytes sent so far.
	n *atomic.Int64
}

// slowCircuitChanBuffer is the buffer used for slow circuits.
const slowCircuitChanBuffer = 4

// newSlowCircuit creates a background goroutine that
// handles the traffic using a slow circuit.
func newSlowCircuit(ctx context.Context, adaptive bool) *slowCircuit {
	sc := &slowCircuit{
		adaptive: adaptive,
		ch:       make(chan *slowCircuitPacket, slowCircuitChanBuffer),
		n:        &atomic.Int64{},
	}
	go sc.loop(ctx)
	return sc
}

// submitOrDrop attempts to submit a packet to the slow circuit.
func (sc *slowCircuit) submitOrDrop(ctx context.Context, nic *NIC, rawPacket []byte) {
	pkt := &slowCircuitPacket{
		nic:       nic,
		rawPacket: rawPacket,
	}
	select {
	case <-ctx.Done():
		// dropped packet
	case sc.ch <- pkt:
		// added to the queue
	default:
		// dropped packet
	}
}

// loop is the main loop of the slow circuit.
func (sc *slowCircuit) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pkt := <-sc.ch:
			sc.delayAndWriteIncoming(ctx, pkt.nic, pkt.rawPacket)
		}
	}
}

// delayAndWriteIncoming delays and then writes the packet
func (sc *slowCircuit) delayAndWriteIncoming(ctx context.Context, nic *NIC, rawPacket []byte) {
	const (
		kilobyte = 1024
		megabyte = kilobyte * kilobyte
	)

	// Implementation note: the following adaptive algorithm is just
	// an experiment to show that it's possible to do that.

	// select the amount of delay to apply
	n := sc.n.Load()
	var delay time.Duration
	switch {
	case !sc.adaptive:
		delay = 40 * time.Millisecond
	case n < 500*kilobyte:
		delay = 0
	case n < megabyte:
		delay = 10 * time.Millisecond
	case n < 2*megabyte:
		delay = 20 * time.Millisecond
	default:
		delay = 40 * time.Millisecond
	}

	// apply the required amount of delay
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		// fallthrough
	}

	// send to the NIC
	_ = nic.WriteIncoming(ctx, rawPacket)

	// update bytes sent stats
	sc.n.Add(int64(len(rawPacket)))
}
