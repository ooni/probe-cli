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

	// DPIStack is the MANDATORY stack to wrap.
	DPIStack
}

var _ DPIStack = &DPIDropTrafficForServerEndpoint{}

// ReadPacket implements DPIStack
func (bs *DPIDropTrafficForServerEndpoint) ReadPacket() ([]byte, error) {
	for {
		rawPacket, err := bs.DPIStack.ReadPacket()
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
		return bs.DPIStack.WritePacket(rawPacket)
	}

	// if the packet matches the offending source, drop it
	if packet.matchSource(bs.ServerProtocol, bs.ServerIPAddress, bs.ServerPort) {
		return nil
	}

	return bs.DPIStack.WritePacket(rawPacket)
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

	// DPIStack is the stack we wrap.
	DPIStack
}

var _ BackboneStack = &DPIDropTrafficForTLSSNI{}

// NewDPIDropTrafficForTLSSNI constructs a [DPIDropTrafficForTLSSNI].
func NewDPIDropTrafficForTLSSNI(stack DPIStack, sni string) *DPIDropTrafficForTLSSNI {
	return &DPIDropTrafficForTLSSNI{
		blackHole: &dpiFlowList{},
		sni:       sni,
		DPIStack:  stack,
	}
}

// ReadPacket implements BackboneStack
func (bs *DPIDropTrafficForTLSSNI) ReadPacket() ([]byte, error) {
	for {
		rawPacket, err := bs.DPIStack.ReadPacket()
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
		return bs.DPIStack.WritePacket(rawPacket)
	}

	// short circuit for UDP packets
	if packet.transportProtocol() != layers.IPProtocolTCP {
		return bs.DPIStack.WritePacket(rawPacket)
	}

	// silently drop the packet if it belongs to a backholed flow
	if bs.blackHole.belongsTo(LinkDirectionRightToLeft, packet) {
		return nil
	}

	// otherwise deliver the packet
	return bs.DPIStack.WritePacket(rawPacket)
}
