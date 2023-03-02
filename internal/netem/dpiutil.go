package netem

//
// DPI: utilities
//

import (
	"sync"

	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DPIStack is the [UNetStack]-like model implemented by DPI code.
type DPIStack interface {
	BackboneStack
	model.UnderlyingNetwork
}

// dpiFlow describes a specific flow. The zero value is invalid; please,
// construct using the [newDPIFlow] constructor.
type dpiFlow struct {
	// protocol is the transport protocol.
	protocol layers.IPProtocol

	// sourceAddress is the source IP address.
	sourceAddress string

	// sourcePort is the source port.
	sourcePort uint16

	// destinationAddress is the destination IP address.
	destinationAddress string

	// destinationPort is the destination port.
	destinationPort uint16
}

// newDPIFlow creates a new DPI flow from the given packet.
func newDPIFlow(packet *dissectedPacket) *dpiFlow {
	f := &dpiFlow{
		protocol:           packet.transportProtocol(),
		sourceAddress:      packet.sourceIPAddress(),
		sourcePort:         0,
		destinationAddress: packet.destinationIPAddress(),
		destinationPort:    0,
	}
	switch {
	case packet.tcp != nil:
		f.sourcePort = uint16(packet.tcp.SrcPort)
		f.destinationPort = uint16(packet.tcp.DstPort)
	case packet.udp != nil:
		f.sourcePort = uint16(packet.udp.SrcPort)
		f.destinationPort = uint16(packet.udp.DstPort)
	default:
		// nothing
	}
	return f
}

// containsPacket returns whether this flow contains the packet
// we're examining also taking the direction into account.
func (f *dpiFlow) containsPacket(direction LinkDirection, packet *dissectedPacket) bool {

	// make sure the protocol is the same and obtain the actual four tuple
	var (
		realSourcePort    uint16
		realSourceAddress = packet.sourceIPAddress()
		realDestPort      uint16
		realDestAddress   = packet.destinationIPAddress()
	)
	switch {
	case f.protocol == layers.IPProtocolTCP && packet.tcp != nil:
		realSourcePort = uint16(packet.tcp.SrcPort)
		realDestPort = uint16(packet.tcp.DstPort)

	case f.protocol == layers.IPProtocolUDP && packet.udp != nil:
		realSourcePort = uint16(packet.udp.SrcPort)
		realDestPort = uint16(packet.udp.DstPort)

	default:
		return false
	}

	// determine the expected four tuple depending on the link direction
	var (
		expectedSourcePort    uint16
		expectedSourceAddress string
		expectedDestPort      uint16
		expectedDestAddress   string
	)
	switch direction {
	case LinkDirectionLeftToRight:
		expectedSourcePort = f.sourcePort
		expectedSourceAddress = f.sourceAddress
		expectedDestPort = f.destinationPort
		expectedDestAddress = f.destinationAddress

	case LinkDirectionRightToLeft:
		expectedSourcePort = f.destinationPort
		expectedSourceAddress = f.destinationAddress
		expectedDestPort = f.sourcePort
		expectedDestAddress = f.sourceAddress

	default:
		return false
	}

	// perform the actual comparison
	return (realSourcePort == expectedSourcePort &&
		realDestPort == expectedDestPort &&
		realSourceAddress == expectedSourceAddress &&
		realDestAddress == expectedDestAddress)
}

// dpiFlowList helps to manage a list of Flows. The zero
// value of this structure is ready to use.
type dpiFlowList struct {
	// mu provides mutual exclusion.
	mu sync.Mutex

	// list contains the list of blackholed flows
	list []*dpiFlow
}

// addFromPacket adds a new flow to the blackhole using data from a packet.
func (bh *dpiFlowList) addFromPacket(packet *dissectedPacket) {
	f := newDPIFlow(packet)
	bh.mu.Lock()
	bh.list = append(bh.list, f)
	bh.mu.Unlock()
}

// belongsTo returns whether the packet belongs to the list.
func (bh *dpiFlowList) belongsTo(direction LinkDirection, packet *dissectedPacket) bool {
	defer bh.mu.Unlock()
	bh.mu.Lock()
	for _, f := range bh.list {
		if f.containsPacket(direction, packet) {
			return true
		}
	}
	return false
}
