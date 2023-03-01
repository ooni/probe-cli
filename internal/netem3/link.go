package netem3

//
// Network link modeling
//

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apex/log"
)

// LinkFrame is a frame encapsulating an IPv4 or IPv6 packet.
type LinkFrame struct {
	// CreationTime is when the frame was created.
	CreationTime time.Time

	// Payload is the IPv4 or IPv6 packet.
	Payload []byte
}

// linkInterfaceID is the unique ID of each link NIC.
var linkInterfaceID = &atomic.Int64{}

// nextLinkInterfaceID returns the next link interface ID.
func nextLinkInterfaceID() string {
	return fmt.Sprintf("eth%d", linkInterfaceID.Add(1))
}

// LinkNIC is the [Link] view of a network interface controller.
type LinkNIC interface {
	// InterfaceName returns the name of the NIC.
	InterfaceName() string

	// ReadFrame reads a frame from the NIC.
	ReadFrame() (*LinkFrame, error)

	// WriteFrame writes a frame to the NIC.
	WriteFrame(frame *LinkFrame) error
}

// LinkDirection is the direction of a link.
type LinkDirection int

// LinkDirectionLeftToRight is the left->right link direction.
const LinkDirectionLeftToRight = LinkDirection(0)

// LinkDirectionRightToLeft is the right->left link direction.
const LinkDirectionRightToLeft = LinkDirection(1)

// LinkConfig contains config for creating a [Link].
type LinkConfig struct {
	// Dump controls whether you want to Dump packets. Should you want
	// to set this flag, you MUST do that before calling Up.
	Dump bool

	// Left is the MANDATORY left NIC device.
	Left LinkNIC

	// LeftToRightBandwidth is the bandwidth in the left->right direction.
	LeftToRightBandwidth Bandwidth

	// LeftToRightDelay is the delay in the left->rigth direction.
	LeftToRightDelay time.Duration

	// Right is the MANDATORY right NIC device.
	Right LinkNIC

	// RightToLeftDelay is the delay in the right->left direction.
	RightToLeftDelay time.Duration

	// RightToLeftBandwidth is the bandwidth in the right->left direction.
	RightToLeftBandwidth Bandwidth
}

// Link models a link between a "left" and a "right" NIC. The zero value
// is invalid; please, use a constructor to create a new instance or manually
// fill all the fields marked as MANDATORY below.
//
// A link is characterized by left-to-right and right-to-left delays, which
// are configured by the [Link] constructors. A link is also characterized
// by a left-to-right and right-to-left bandwidths. In principle, the performance
// could be a property of the NIC or of the link. But it seems more accurate to
// attach it to the link, because of 10/100/1000 Ethernet cards. At the end
// of the day, it is the link that determines the bandwidth.
//
// Do not assume that setting a specific bandwidth and delay is going to yield
// very accurate results like this was a good simulator you could use for writing
// papers about TCP performance. We did not write this code with that use case
// in mind. Rather, here the objective is to be able to detect dramatic throttling
// cases where the speed drops of a ~10x factor across test cases.
//
// Once you created a link, it will not forward traffic between its left
// and right NICs until you call the [Link.Up] method.
//
// After you have called [Link.Up], the left-to-right fowarding works
// as depicted by the following diagram:
//
//	.------.
//	| Left | ---> ReadOutgoing ---> EmulateTXRXDelay ---> <<new goroutine>>
//	'------'                                                     |
//	                                                             V
//	                                    .-------------- EmulateLeftToRightDelay
//	                                    |
//	                                    |
//	                                    V         true
//	                                dpi.Divert ----------> Packet handled by dpi
//	                                    |                     (maybe dropped)
//	                                    | false
//	                                    |
//	.-------.                           V
//	| Right | <--- WriteIncoming <--- dpi.Delay
//	'-------'                           (maybe throttling)
//
// We emulate the TXRX delay of the link in the same goroutine in which we
// read the packet, because that is how sending and transmitting over a channel
// looks like. After that, we fork off a packet-specific goroutine, which is
// responsible of emulating the propagation delay. We must do this because
// otherwise we could not have multiple packets in flight.
//
// Note that we call the dpi.Divert hook after emulating the delay of the
// link. When the hook returns true, we stop caring about the packet. When
// it retuns false, we call the dpi.Delay hook, which does not divert the
// packet but allows to implement throttling. Finally, we deliver the packet
// to the right NIC by calling its WriteIncoming method.
//
// The right-to-left direction works similarly, except that we emulate the
// right-to-left delay after dpi.Divert. We do this to model the DPI device
// as generally close the the user, which lives on the left.
//
// Typically, one uses [Backbone] to manage several [Link]s and implement
// routing. In such a case it is worth remembering the following:
//
//   - when you're modeling a client stub network, the left-to-right
//     direction flows from the client to the backbone;
//
//   - when you're modeling a server stub network, the left-to-right
//     direction flows from the server to the backbone.
//
// In order words, the backbone is always on the right-hand size of
// both client and server stub networks. This fact is also documented
// by the documentation of [Backbone].
type Link struct {
	shutdown context.CancelFunc
	wg       *sync.WaitGroup
}

// NewLink creates a new [Link] instance and spawns goroutines for forwarding
// traffic between the left and the right [LinkNIC]. You MUST call [Link.Close] to
// stop these goroutines when you are done with the [Link].
func NewLink(config *LinkConfig) *Link {
	// create context for interrupting the [Link].
	ctx, cancel := context.WithCancel(context.Background())

	// create wait group to synchronize with [Link.Close]
	wg := &sync.WaitGroup{}

	// forward in the left->right direction.
	wg.Add(1)
	go linkForward(
		ctx,
		config.Dump,
		LinkDirectionLeftToRight,
		config.Left,
		config.Right,
		config.LeftToRightBandwidth,
		config.LeftToRightDelay,
		wg,
	)

	// forward in the right->left direction.
	wg.Add(1)
	go linkForward(
		ctx,
		config.Dump,
		LinkDirectionRightToLeft,
		config.Right,
		config.Left,
		config.RightToLeftBandwidth,
		config.RightToLeftDelay,
		wg,
	)

	link := &Link{
		shutdown: cancel,
		wg:       wg,
	}
	return link
}

// Close closes the [Link].
func (lnk *Link) Close() error {
	lnk.shutdown()
	lnk.wg.Wait()
	return nil
}

// readableLinkNIC is a read-only [LinkNIC]
type readableLinkNIC interface {
	InterfaceName() string
	ReadFrame() (*LinkFrame, error)
}

// writeableLinkNIC is a write-only [LinkNIC]
type writeableLinkNIC interface {
	InterfaceName() string
	WriteFrame(frame *LinkFrame) error
}

// linkForward forwards traffic between reader and writer
func linkForward(
	ctx context.Context,
	dump bool,
	direction LinkDirection,
	reader readableLinkNIC,
	writer writeableLinkNIC,
	bw Bandwidth,
	delay time.Duration,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		// read a frame from the source NIC
		frame, err := reader.ReadFrame()
		if err != nil {
			log.Warnf("netem: linkForward: WriteFrame: %s", err.Error())
			return
		}

		// dump the frame
		maybeDumpPacket(dump, reader.InterfaceName()+"->", frame.Payload)

		// compute the transmission delay
		d := linkComputeTXDelay(bw, len(frame.Payload))

		// add the propagation delay
		d += delay

		// compute the frame arrival deadline
		arrival := frame.CreationTime.Add(d)

		// if needed sleep to deliver the packet at the right time
		if err := linkMaybeEmulateDelay(ctx, -time.Since(arrival)); err != nil {
			log.Warnf("netem: linkForward: linkMaybeEmulateDelay: %s", err.Error())
			return
		}

		// dump the frame
		maybeDumpPacket(dump, writer.InterfaceName()+"<-", frame.Payload)

		// write the frame to the destination NIC
		if err := writer.WriteFrame(frame); err != nil {
			log.Warnf("netem: linkForward: ReadFrame: %s", err.Error())
			return
		}
	}
}

// linkComputeTXDelay computes the TX delay for a given packet. This
// function returns zero in case we don't need to set a delay.
func linkComputeTXDelay(speed Bandwidth, count int) (out time.Duration) {
	if speed > 0 && count > 0 {
		out = (time.Duration(count) * 8 * time.Second) / time.Duration(speed)
	}
	return
}

// linkMaybeEmulateDelay possibly adds delay to the transmission.
func linkMaybeEmulateDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
