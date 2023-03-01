package netem

//
// Composable deep-packet-inspection rules
//

import (
	"context"
	"sync"

	"github.com/google/gopacket/layers"
)

// DPINone is a [LinkDPIEngine] that does nothing.
type DPINone struct{}

var _ LinkDPIEngine = &DPINone{}

// Divert implements LinkDPIEngine
func (*DPINone) Divert(
	ctx context.Context,
	direction LinkDirection,
	source *NIC,
	dest *NIC,
	rawPacket []byte,
) bool {
	return false
}

// DPIDropTrafficForServerEndpoint is a [LinkDPIEngine] that drops all
// the traffic towards a given server endpoint. The zero value is invalid;
// please fill all the fields marked as MANDATORY.
type DPIDropTrafficForServerEndpoint struct {
	// Direction is the MANDATORY packets flow direction. Use
	// [LinkDirectionLeftToRight] when you are installing this
	// DPI rule on the client side; use [LinkDirectionRightToLeft]
	// when you are installing it on the server side.
	Direction LinkDirection

	// ServerIPAddress is the MANDATORY server endpoint IP address.
	ServerIPAddress string

	// ServerPort is the MANDATORY server endpoint port.
	ServerPort uint16

	// ServerProtocol is the MANDATORY server endpoint protocol.
	ServerProtocol layers.IPProtocol
}

var _ LinkDPIEngine = &DPIDropTrafficForServerEndpoint{}

// Divert implements LinkDPIEngine
func (e *DPIDropTrafficForServerEndpoint) Divert(
	ctx context.Context,
	direction LinkDirection,
	source *NIC,
	dest *NIC,
	rawPacket []byte,
) bool {
	// check whether packet is flowing in the expected direction
	if direction != e.Direction {
		return false // wrong direction, let it flow
	}

	// parse the packet
	packet, err := dissect(rawPacket)
	if err != nil {
		return false // we don't know how to handle this packet, let it flow
	}

	// it's our packet if it maches the expected destination
	return packet.matchDestination(e.ServerProtocol, e.ServerIPAddress, e.ServerPort)
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

// DPIDropTrafficForTLSSNI is a [LinkDPIEngine] that drops all
// the traffic after it sees a given TLS SNI. The zero value is
// invalid; construct using [NewDPIDropTrafficForTLSSNI].
//
// You MUST insert this DPI filter on the client side (which is
// what this library encourages doing anyway).
type DPIDropTrafficForTLSSNI struct {
	// blackHole contains information about blackholed flows.
	blackHole *dpiFlowList

	// sni is the offending SNI.
	sni string
}

var _ LinkDPIEngine = &DPIDropTrafficForTLSSNI{}

// NewDPIDropTrafficForTLSSNI constructs a [DPIDropTrafficForTLSSNI].
func NewDPIDropTrafficForTLSSNI(sni string) *DPIDropTrafficForTLSSNI {
	return &DPIDropTrafficForTLSSNI{
		blackHole: &dpiFlowList{},
		sni:       sni,
	}
}

// Divert implements LinkDPIEngine
func (e *DPIDropTrafficForTLSSNI) Divert(
	ctx context.Context,
	direction LinkDirection,
	source *NIC,
	dest *NIC,
	rawPacket []byte,
) bool {
	// parse the packet
	packet, err := dissect(rawPacket)
	if err != nil {
		return false
	}

	// short circuit for UDP packets
	if packet.transportProtocol() != layers.IPProtocolTCP {
		return false
	}

	// check whether this packet belongs to a blackholed flow
	if e.blackHole.belongsTo(direction, packet) {
		return true
	}

	// short circuit for packets flowing in the wrong direction
	if direction != LinkDirectionLeftToRight {
		return false
	}

	// try to obtain the SNI and stop processing if it's not the offending one
	sni, err := packet.parseTLSServerName()
	if err != nil {
		return false
	}
	if sni != e.sni {
		return false
	}

	// we must prevent this packet from routing
	e.blackHole.addFromPacket(packet)
	return true
}

// dpiThrottledFlow is a flow that is being throttled. The zero
// value is invalid; use [newDPIThrottledFlow] to construct.
type dpiThrottledFlow struct {
	// circuit is the slow circuit to use.
	circuit *slowCircuit

	// flow is the underlying flow
	flow *dpiFlow
}

// newDPIThrottledFlow constructs a [dpiThrottledFlow]. This function will start
// goroutines taking care of throttling. They will run as long as ctx is not canceled.
func newDPIThrottledFlow(ctx context.Context, packet *dissectedPacket) *dpiThrottledFlow {
	return &dpiThrottledFlow{
		circuit: newSlowCircuit(ctx, true),
		flow:    newDPIFlow(packet),
	}
}

// dpiThrottledFlowList is a list of throttled flows. The zero
// value of this structure is ready to use.
type dpiThrottledFlowList struct {
	// list is the list of flows being throttled.
	list []*dpiThrottledFlow

	// mu provides mutual exclusion
	mu sync.Mutex
}

// addFromPacket creates a new throttling flow and adds it to the
// list. This function creates I/O goroutines to throttle this flow,
// which will run until the given context is canceled.
func (l *dpiThrottledFlowList) addFromPacket(ctx context.Context, packet *dissectedPacket) {
	tf := newDPIThrottledFlow(ctx, packet)
	defer l.mu.Unlock()
	l.mu.Lock()
	l.list = append(l.list, tf)
}

// lookup returns the slow circuit throttling the flow to which this
// packet belongs, or false in case there is none.
func (bh *dpiThrottledFlowList) lookup(
	direction LinkDirection,
	packet *dissectedPacket,
) (*slowCircuit, bool) {
	defer bh.mu.Unlock()
	bh.mu.Lock()
	for _, f := range bh.list {
		if f.flow.containsPacket(direction, packet) {
			return f.circuit, true
		}
	}
	return nil, false
}

// DPIThrottleTrafficForTLSSNI is a [LinkDPIEngine] that throttles
// traffic after it sees a given TLS SNI. The zero value is
// invalid; construct using [NewDPIThrottleTrafficForTLSSNI].
//
// You MUST insert this DPI filter on the client side (which is
// what this library encourages doing anyway).
type DPIThrottleTrafficForTLSSNI struct {
	// slowed contains information about slowed-down flows.
	slowed *dpiThrottledFlowList

	// sni is the offending SNI.
	sni string
}

var _ LinkDPIEngine = &DPIThrottleTrafficForTLSSNI{}

// NewDPIThrottleTrafficForTLSSNI constructs a [DPIThrottleTrafficForTLSSNI].
func NewDPIThrottleTrafficForTLSSNI(sni string) *DPIThrottleTrafficForTLSSNI {
	return &DPIThrottleTrafficForTLSSNI{
		slowed: &dpiThrottledFlowList{},
		sni:    sni,
	}
}

// Divert implements LinkDPIEngine
func (e *DPIThrottleTrafficForTLSSNI) Divert(
	ctx context.Context,
	direction LinkDirection,
	source *NIC,
	dest *NIC,
	rawPacket []byte,
) bool {
	// parse the packet
	packet, err := dissect(rawPacket)
	if err != nil {
		return false
	}

	// short circuit for UDP packets
	if packet.transportProtocol() != layers.IPProtocolTCP {
		return false
	}

	// packets flowing from left to right (client to server) are
	// interesting to spot the TLS SNI and choose whether the throttle,
	// while packets in the other directions could be throttled if
	// they happen to belong to a slowed-down flow.
	if direction != LinkDirectionLeftToRight {
		circuit, good := e.slowed.lookup(direction, packet)
		if good {
			circuit.submitOrDrop(ctx, dest, rawPacket)
			return true // packet routed over a slow circuit or dropped
		}
		return false
	}

	// try to obtain the SNI and stop processing if it's not the offending one
	sni, err := packet.parseTLSServerName()
	if err != nil {
		return false
	}
	if sni != e.sni {
		return false
	}

	// create the slow circuit but otherwise don't disturb this packet
	e.slowed.addFromPacket(ctx, packet)
	return false
}
