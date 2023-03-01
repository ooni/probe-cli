package netem

//
// Network link modeling
//

import (
	"context"
	"errors"
	"time"

	"github.com/apex/log"
)

// LinkDPIEngine is the [Link] view of a DPI engine. See the documentation
// of [Link] for more information about the overall topology.
type LinkDPIEngine interface {
	// Divert allows a [LinkDPIEngine] to prevent a [Link] from forwarding a
	// given rawPacket. To this end, [Divert] must return true. See the
	// documentation of [Link] for more information.
	Divert(
		ctx context.Context,
		direction LinkDirection,
		source *NIC,
		dest *NIC,
		rawPacket []byte,
	) bool
}

// LinkDirection is the direction of a link.
type LinkDirection int

// LinkDirectionLeftToRight is the left->right link direction.
const LinkDirectionLeftToRight = LinkDirection(0)

// LinkDirectionRightToLeft is the right->left link direction.
const LinkDirectionRightToLeft = LinkDirection(1)

// Link models a link between a "left" and a "right" NIC. The zero value
// is invalid; please, use a constructor to create a new instance or manually
// fill all the fields marked as MANDATORY below.
//
// A link is characterized by left-to-right and right-to-left delays, which
// are configured by the [Link] constructors. Those delays do not allow
// for accurate modeling of network performance. However, we have calibrated
// specific delays such that we can construct links with rougly one order
// of magnitude performance difference between each other.
//
// Once you created a link, it will not forward traffic between its left
// and right NICs until you call the [Link.Up] method.
//
// After you have called [Link.Up], the left-to-right fowarding works
// as depicted by the following diagram:
//
//	.------.
//	| Left | ---> ReadOutgoing ---> EmulateLeftToRightDelay
//	'------'                            |
//	                                    |
//	                                    |
//	                                    V         true
//	                                dpi.Divert ----------> Packet handled by dpi
//	                                    |
//	                                    | false
//	                                    |
//	              .-------.             V
//	              | Right | <---- WriteIncoming
//	              '-------'
//
// That is, we call the dpi.Divert hook after emulating the delay of the
// link. When the hook returns true, we stop caring about the packet. When
// it retuns false, we pass the packet to the right NIC.
//
// The right-to-left direction works similarly.
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
	// DPI is the MANDATORY DPI engine to use.
	DPI LinkDPIEngine

	// Dump controls whether you want to Dump packets. Should you want
	// to set this flag, you MUST do that before calling Up.
	Dump bool

	// Left is the MANDATORY left NIC device.
	Left *NIC

	// LeftToRightDelay is the delay in the left->rigth direction.
	LeftToRightDelay time.Duration

	// Right is the MANDATORY right NIC device.
	Right *NIC

	// RightToLeftDelay is the delay in the right->left direction.
	RightToLeftDelay time.Duration
}

// LinkFactory the signature of the function that creates a [Link].
type LinkFactory func(left, right *NIC, dpi LinkDPIEngine) *Link

// NewLinkVerbose wraps a LinkFactory such that you end up creating a [Link]
// that dumps packets as they leave and enter into [NIC]s.
func NewLinkVerbose(factory LinkFactory) LinkFactory {
	return func(left, right *NIC, dpi LinkDPIEngine) *Link {
		link := factory(left, right, dpi)
		link.Dump = true
		return link
	}
}

// NewLinkFastest returns the fastest possible [Link] without any delay.
func NewLinkFastest(left, right *NIC, dpi LinkDPIEngine) *Link {
	return &Link{
		DPI:              dpi,
		Left:             left,
		LeftToRightDelay: 0,
		Right:            right,
		RightToLeftDelay: 0,
	}
}

// NewLinkMedium returns a slower [Link] than [NewLinkFastest]. We calibrated
// the settings to obtain around 8 Mbit/s when using DASH.
func NewLinkMedium(left, right *NIC, dpi LinkDPIEngine) *Link {
	return &Link{
		DPI:              dpi,
		Left:             left,
		LeftToRightDelay: time.Millisecond,
		Right:            right,
		RightToLeftDelay: time.Millisecond,
	}
}

// NewLinkSlowest returns a slower [Link] than [NewLinkMedium]. We calibrated
// the settings to ontain around 400 kbit/s when using DASH.
func NewLinkSlowest(left, right *NIC, dpi LinkDPIEngine) *Link {
	return &Link{
		DPI:              dpi,
		Left:             left,
		LeftToRightDelay: 20 * time.Millisecond,
		Right:            right,
		RightToLeftDelay: 20 * time.Millisecond,
	}
}

// Up spawns goroutines forwarding traffic between the two NICs until the given context
// expires or is cancelled. You MUST NOT call this function more than once.
func (l *Link) Up(ctx context.Context) {
	// left->right
	go l.forward(
		ctx,
		LinkDirectionLeftToRight,
		l.Left,
		l.Right,
		l.LeftToRightDelay,
	)

	// right->left
	go l.forward(
		ctx,
		LinkDirectionRightToLeft,
		l.Right,
		l.Left,
		l.RightToLeftDelay,
	)
}

// forward forwards traffic between two TUNs.
func (l *Link) forward(
	ctx context.Context,
	direction LinkDirection,
	reader *NIC,
	writer *NIC,
	delay time.Duration,
) {
	log.Infof("netem: link %s -> %s up", reader.name, writer.name)
	defer log.Infof("netem: link %s -> %s down", reader.name, writer.name)

	for {
		// read from the reader NIC
		rawPacket, err := reader.ReadOutgoing(ctx)
		if err != nil {
			log.Warnf("netem: link.forward: %s", ctx.Err().Error())
			return
		}

		// dump before emulating delay for pretty obvious reasons
		maybeDumpPacket(l.Dump, reader.name+"->", rawPacket)

		// deliver this packet in the background
		go l.deliverPacket(ctx, direction, reader, writer, delay, rawPacket)
	}
}

// deliverPacket delivers a single packet.
func (l *Link) deliverPacket(
	ctx context.Context,
	direction LinkDirection,
	reader *NIC,
	writer *NIC,
	delay time.Duration,
	rawPacket []byte,
) {
	// emulate the delay
	if err := linkMaybeEmulateDelay(ctx, delay); err != nil {
		log.Warnf("netem: link.deliverPacket: %s", err.Error())
		return
	}

	// possibly divert the packet through the dpi engine
	if l.DPI != nil && l.DPI.Divert(ctx, direction, reader, writer, rawPacket) {
		return
	}

	// only dump the packet entering the interface after we know
	// it has not been diverted by the DPI
	maybeDumpPacket(l.Dump, writer.name+"<-", rawPacket)

	// write to the writer NIC
	if err := writer.WriteIncoming(ctx, rawPacket); err != nil {
		if !errors.Is(err, ErrNICBufferFull) {
			log.Warnf("netem: link.deliverPacket: %s", err.Error())
		}
		return
	}
}

// linkMaybeEmulateDelay adds delay to the transmission.
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
