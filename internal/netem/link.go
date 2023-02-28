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

// LinkDPIEngine is the [Link] view of a DPI engine.
type LinkDPIEngine interface {
	// Divert is called by a [Link] right before emitting the
	// given rawPacket on the given dest interface. The DPIEngine
	// should return true to notify the [Link] that it will deliver
	// the packet. Otherwise, the [Link] will deliver the packet
	// as usual. The direction argument provides the packet direction
	// where "left" is the client and "right" the server. The source
	// argument allows responding immediately to the client.
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

// Link models a link between two NICs. By convention, we call these
// two NICs "the left NIC" and "the right NIC". A [Link] is characterized by
// a left-to-right propagation delay and bandwidth, as well as by a
// right-to-left propagation delay and bandwidth. The zero value of this
// structure is invalid; to construct, you MUST fill all the MANDATORY
// fields. By itself, a link does not forward traffic in either direction,
// until you call [Link.Run] in a background goroutine.
type Link struct {
	// DPI is the OPTIONAL DPI engine. If you need to set this field,
	// you MUST do that BEFORE calling the [Link.Up] function.
	DPI LinkDPIEngine

	// Left is the MANDATORY left NIC device.
	Left *NIC

	// LeftToRightDelay is the OPTIONAL delay in the left->rigth direction.
	LeftToRightDelay time.Duration

	// Right is the MANDATORY right NIC device.
	Right *NIC

	// RightToLeftDelay is the OPTIONAL delay in the right->left direction.
	RightToLeftDelay time.Duration
}

// LinkFactory the signature of the function that creates a [Link].
type LinkFactory func(left, right *NIC) *Link

// NewLinkFastest returns the fastest possible [Link] without any delay.
func NewLinkFastest(left, right *NIC) *Link {
	return &Link{
		DPI:              nil,
		Left:             left,
		LeftToRightDelay: 0,
		Right:            right,
		RightToLeftDelay: 0,
	}
}

// NewLinkMedium returns a slower [Link] than [NewLinkFastest]. We calibrated
// the settings to obtain around 8 Mbit/s when using DASH.
func NewLinkMedium(left, right *NIC) *Link {
	return &Link{
		DPI:              nil,
		Left:             left,
		LeftToRightDelay: time.Millisecond,
		Right:            right,
		RightToLeftDelay: time.Millisecond,
	}
}

// NewLinkSlowest returns a slower [Link] than [NewLinkMedium]. We calibrated
// the settings to ontain around 400 kbit/s when using DASH.
func NewLinkSlowest(left, right *NIC) *Link {
	return &Link{
		DPI:              nil,
		Left:             left,
		LeftToRightDelay: 20 * time.Millisecond,
		Right:            right,
		RightToLeftDelay: 20 * time.Millisecond,
	}
}

// Up spawns goroutines forwarding traffic between the two NICs until the given context
// expires or is cancelled. You MUST NOT call this function more than once.
func (l *Link) Up(ctx context.Context, dump bool) {
	// left->right
	go l.linkForward(
		ctx,
		LinkDirectionLeftToRight,
		l.Left,
		l.Right,
		l.LeftToRightDelay,
		dump,
	)

	// right->left
	go l.linkForward(
		ctx,
		LinkDirectionRightToLeft,
		l.Right,
		l.Left,
		l.RightToLeftDelay,
		dump,
	)
}

// linkForward forwards traffic between two TUNs.
func (l *Link) linkForward(
	ctx context.Context,
	direction LinkDirection,
	reader *NIC,
	writer *NIC,
	delay time.Duration,
	dump bool,
) {
	log.Infof("netem: link %s -> %s up", reader.name, writer.name)
	defer log.Infof("netem: link %s -> %s down", reader.name, writer.name)

	for {
		// read from the reader NIC
		rawPacket, err := reader.ReadOutgoing(ctx)
		if err != nil {
			log.Warnf("netem: linkForward: %s", ctx.Err().Error())
			return
		}

		maybeDumpPacket(dump, reader.name+"->", rawPacket)

		// emulate the delay
		if err := linkMaybeEmulateDelay(ctx, delay); err != nil {
			log.Warnf("netem: linkForward: %s", err.Error())
			return
		}

		maybeDumpPacket(dump, writer.name+"<-", rawPacket)

		// possibly divert the packet through the DPI engine
		if l.DPI != nil && l.DPI.Divert(ctx, direction, reader, writer, rawPacket) {
			continue
		}

		// write to the writer NIC
		if err := writer.WriteIncoming(ctx, rawPacket); err != nil {
			log.Warnf("netem: linkForward: %s", ctx.Err().Error())
			if !errors.Is(err, ErrNICBufferFull) {
				return
			}
		}
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
