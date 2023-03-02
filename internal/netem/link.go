package netem

//
// Network link modeling
//

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

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

	// ReadPacket should return the next packet to send over the [Link].
	ReadPacket() ([]byte, error)

	// WritePacket is called by a [Link] to deliver a packet.
	WritePacket(packet []byte) error
}

// LinkDirection is the direction of a link.
type LinkDirection int

// LinkDirectionLeftToRight is the left->right link direction.
const LinkDirectionLeftToRight = LinkDirection(0)

// LinkDirectionRightToLeft is the right->left link direction.
const LinkDirectionRightToLeft = LinkDirection(1)

// LinkConfig contains config for creating a [Link].
type LinkConfig struct {
	// LeftToRightPLR is the packet-loss rate in the left->right direction.
	LeftToRightPLR float64

	// LeftToRightDelay is the delay in the left->rigth direction.
	LeftToRightDelay time.Duration

	// RightToLeftDelay is the delay in the right->left direction.
	RightToLeftDelay time.Duration

	// RightToLeftPLR is the packet-loss rate in the right->left direction.
	RightToLeftPLR float64
}

// Link models a link between a "left" and a "right" NIC. The zero value
// is invalid; please, use a constructor to create a new instance or manually
// fill all the fields marked as MANDATORY below.
//
// A link is characterized by left-to-right and right-to-left delays, which
// are configured by the [Link] constructors. A link is also characterized
// by a left-to-right and right-to-left packet loss rate (PLR).
//
// Do not assume that setting a specific PLR and delay is going to yield
// very accurate results like this was a good simulator you could use for writing
// papers about TCP performance. We did not write this code with that use case
// in mind. Rather, here the objective is to be able to detect dramatic throttling
// cases where the speed drops of a ~10x factor across test cases.
//
// Once you created a link, it will immediately start to forward traffic
// until you call [Link.Close] to shut it down.
//
// We create a goroutine for each possible packet in flight in each of
// the two directions. The following diagram illustrates what happens
// when a goroutine moves a packet from left to right:
//
//	.------.
//	| Left | ---> ReadPacket ---> Apply PLR policy ---> <<drop>>
//	'------'                             |
//	                                     V
//	                                  <<keep>>
//	                                     |
//	                                     V
//	                             Apply delay policy
//	                                     |
//	.-------.                            V
//	| Right | <-------------------- WritePacket
//	'-------'
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
	// shutdown allows us to shutdown a link
	shutdown context.CancelFunc

	// wg allows us to wait for the background goroutines
	wg *sync.WaitGroup
}

// NewLink creates a new [Link] instance and spawns goroutines for forwarding
// traffic between the left and the right [LinkNIC]. You MUST call [Link.Close] to
// stop these goroutines when you are done with the [Link].
func NewLink(left, right LinkNIC, config *LinkConfig) *Link {
	// create context for interrupting the [Link].
	ctx, cancel := context.WithCancel(context.Background())

	// create wait group to synchronize with [Link.Close]
	wg := &sync.WaitGroup{}

	// create link losses managers
	leftLLM := newLinkLossesManager(config.LeftToRightPLR)
	rightLLM := newLinkLossesManager(config.RightToLeftPLR)

	// this is the maximum number of packets in flight per direction,
	// which limits the maximum congestion window
	const maxInFlight = 1000

	// forward in the left->right direction.
	for i := 0; i < maxInFlight; i++ {
		wg.Add(1)
		go linkForward(
			ctx,
			leftLLM,
			LinkDirectionLeftToRight,
			left,
			right,
			config.LeftToRightDelay,
			wg,
		)
	}

	// forward in the right->left direction.
	for i := 0; i < maxInFlight; i++ {
		wg.Add(1)
		go linkForward(
			ctx,
			rightLLM,
			LinkDirectionRightToLeft,
			right,
			left,
			config.RightToLeftDelay,
			wg,
		)
	}

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
	ReadPacket() ([]byte, error)
}

// writeableLinkNIC is a write-only [LinkNIC]
type writeableLinkNIC interface {
	InterfaceName() string
	WritePacket(packet []byte) error
}

// linkForward models the life of a packet in flight
func linkForward(
	ctx context.Context,
	llm *linkLossesManager,
	direction LinkDirection,
	reader readableLinkNIC,
	writer writeableLinkNIC,
	oneWayDelay time.Duration,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		// read a frame from the source NIC
		rawPacket, err := reader.ReadPacket()
		if err != nil {
			return
		}

		// drop the packet according to the PLR policy
		if llm.shouldDrop() {
			continue
		}

		// honour the one-way propagation delay
		select {
		case <-ctx.Done():
			return
		case <-time.After(oneWayDelay):
		}

		// write the frame to the destination NIC
		if err := writer.WritePacket(rawPacket); err != nil {
			return
		}
	}
}

// linkLossesManager manages losses on the link. The zero value
// is invalid, use [newLinkLossesManager] to construct.
type linkLossesManager struct {
	// mu provides mutual exclusion
	mu sync.Mutex

	// rng is the random number generator.
	rng *rand.Rand

	// targetPLR is the target PLR.
	targetPLR float64
}

// newLinkLossesManager creates a new [linkLossesManager].
func newLinkLossesManager(targetPLR float64) *linkLossesManager {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &linkLossesManager{
		mu:        sync.Mutex{},
		rng:       rng,
		targetPLR: targetPLR,
	}
}

// shouldDrop returns true if this packet should be dropped.
func (llm *linkLossesManager) shouldDrop() bool {
	defer llm.mu.Unlock()
	llm.mu.Lock()
	return llm.rng.Float64() < llm.targetPLR
}
