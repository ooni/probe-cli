package netem

//
// DPI: rules to drop packets
//

import (
	"github.com/google/gopacket/layers"
)

// DPIDropTrafficForServerEndpoint is a [LinkDPIEngine] that drops all
// the traffic towards a given server endpoint. The zero value is invalid;
// please fill all the fields marked as MANDATORY.
type DPIDropTrafficForServerEndpoint struct {
	// ServerIPAddress is the MANDATORY server endpoint IP address.
	ServerIPAddress string

	// ServerPort is the MANDATORY server endpoint port.
	ServerPort uint16

	// ServerProtocol is the MANDATORY server endpoint protocol.
	ServerProtocol layers.IPProtocol

	// Stack is the MANDATORY stack to wrap.
	Stack BackboneStack
}

var _ BackboneStack = &DPIDropTrafficForServerEndpoint{}

// InterfaceName implements BackboneStack
func (bs *DPIDropTrafficForServerEndpoint) InterfaceName() string {
	return bs.Stack.InterfaceName()
}

// ReadPacket implements BackboneStack
func (bs *DPIDropTrafficForServerEndpoint) ReadPacket() ([]byte, error) {
	for {
		rawPacket, err := bs.Stack.ReadPacket()
		if err != nil {
			return nil, err
		}

		// parse the packet
		packet, err := dissectPacket(rawPacket)
		if err != nil {
			return rawPacket, nil
		}

		// if the packet matches the offending destination, silently drop it
		if packet.matchDestination(bs.ServerProtocol, bs.ServerIPAddress, bs.ServerPort) {
			continue
		}

		// otherwise return the packet
		return rawPacket, nil
	}
}

// WritePacket implements BackboneStack
func (bs *DPIDropTrafficForServerEndpoint) WritePacket(rawPacket []byte) error {
	// parse the packet
	packet, err := dissectPacket(rawPacket)
	if err != nil {
		return bs.Stack.WritePacket(rawPacket)
	}

	// if the packet matches the offending source, drop it
	if packet.matchSource(bs.ServerProtocol, bs.ServerIPAddress, bs.ServerPort) {
		return nil
	}

	return bs.Stack.WritePacket(rawPacket)
}

// Close implements BackboneStack
func (bs *DPIDropTrafficForServerEndpoint) Close() error {
	return bs.Stack.Close()
}

// IPAddress implements BackboneStack
func (bs *DPIDropTrafficForServerEndpoint) IPAddress() string {
	return bs.Stack.IPAddress()
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

	// stack is the [BackboneStack] we wrap.
	stack BackboneStack
}

var _ BackboneStack = &DPIDropTrafficForTLSSNI{}

// NewDPIDropTrafficForTLSSNI constructs a [DPIDropTrafficForTLSSNI].
func NewDPIDropTrafficForTLSSNI(stack BackboneStack, sni string) *DPIDropTrafficForTLSSNI {
	return &DPIDropTrafficForTLSSNI{
		blackHole: &dpiFlowList{},
		sni:       sni,
		stack:     stack,
	}
}

// InterfaceName implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) InterfaceName() string {
	return bs.stack.InterfaceName()
}

// ReadPacket implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) ReadPacket() ([]byte, error) {
	for {
		rawPacket, err := bs.stack.ReadPacket()
		if err != nil {
			return nil, err
		}

		// parse the packet
		packet, err := dissectPacket(rawPacket)
		if err != nil {
			return rawPacket, nil
		}

		// short circuit for UDP packets
		if packet.transportProtocol() != layers.IPProtocolTCP {
			return rawPacket, nil
		}

		// drop this packet if it belongs to a blackholed flow
		if bs.blackHole.belongsTo(LinkDirectionLeftToRight, packet) {
			continue
		}

		// try to obtain the SNI
		sni, err := packet.parseTLSServerName()
		if err != nil {
			return rawPacket, nil
		}

		// if the SNI is the offending SNI, insert the flow
		// into the backhole and ignore the packet.
		if sni == bs.sni {
			bs.blackHole.addFromPacket(packet)
			continue
		}

		// otherwise return the packet
		return rawPacket, nil
	}
}

// WritePacket implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) WritePacket(rawPacket []byte) error {
	// parse the packet
	packet, err := dissectPacket(rawPacket)
	if err != nil {
		return bs.stack.WritePacket(rawPacket)
	}

	// short circuit for UDP packets
	if packet.transportProtocol() != layers.IPProtocolTCP {
		return bs.stack.WritePacket(rawPacket)
	}

	// silently drop the packet if it belongs to a backholed flow
	if bs.blackHole.belongsTo(LinkDirectionRightToLeft, packet) {
		return nil
	}

	// otherwise deliver the packet
	return bs.stack.WritePacket(rawPacket)
}

// Close implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) Close() error {
	return bs.stack.Close()
}

// IPAddress implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) IPAddress() string {
	return bs.stack.IPAddress()
}
