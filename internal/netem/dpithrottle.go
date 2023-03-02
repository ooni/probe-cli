package netem

//
// DPI: rules to throttle flows
//

import (
	"github.com/google/gopacket/layers"
)

// DPIThrottleTrafficForTLSSNI is a [LinkDPIEngine] that throttles
// traffic after it sees a given TLS SNI. The zero value is
// invalid; construct using [NewDPIThrottleTrafficForTLSSNI].
type DPIThrottleTrafficForTLSSNI struct {
	// plrm manages the PLR for throttled packets.
	plrm *linkLossesManager

	// slowed contains information about slowed-down flows.
	slowed *dpiFlowList

	// sni is the offending SNI.
	sni string

	// stack is the [BackboneStack] we wrap.
	stack BackboneStack
}

var _ BackboneStack = &DPIThrottleTrafficForTLSSNI{}

// NewDPIThrottleTrafficForTLSSNI constructs a [DPIThrottleTrafficForTLSSNI].
func NewDPIThrottleTrafficForTLSSNI(
	stack BackboneStack,
	sni string,
	targetPLR float64,
) *DPIThrottleTrafficForTLSSNI {
	return &DPIThrottleTrafficForTLSSNI{
		plrm:   newLinkLossesManager(targetPLR),
		slowed: &dpiFlowList{},
		sni:    sni,
		stack:  stack,
	}
}

// InterfaceName implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) InterfaceName() string {
	return bs.stack.InterfaceName()
}

// ReadPacket implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) ReadPacket() ([]byte, error) {
	for {
		rawPacket, err := bs.stack.ReadPacket()
		if err != nil {
			return nil, err
		}

		// parse the packet
		packet, err := dissectPacket(rawPacket)
		if err != nil {
			return nil, err
		}

		// short circuit for UDP packets
		if packet.transportProtocol() != layers.IPProtocolTCP {
			return rawPacket, nil
		}

		// try to obtain the SNI
		sni, err := packet.parseTLSServerName()
		if err != nil {
			return rawPacket, nil
		}

		// if the packet is not offending, deliver it
		if sni != bs.sni {
			return rawPacket, nil
		}

		// regiser as offending and continue processing packets
		bs.slowed.addFromPacket(packet)
	}
}

// WritePacket implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) WritePacket(rawPacket []byte) error {
	// parse the packet
	packet, err := dissectPacket(rawPacket)
	if err != nil {
		return bs.stack.WritePacket(rawPacket)
	}

	// if this packet is slowed down check whether we should drop it
	if bs.slowed.belongsTo(LinkDirectionRightToLeft, packet) {
		if bs.plrm.shouldDrop() {
			return nil
		}
		// fallthrough
	}

	// deliver the packet
	return bs.stack.WritePacket(rawPacket)
}

// Close implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) Close() error {
	return bs.stack.Close()
}

// IPAddress implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) IPAddress() string {
	return bs.stack.IPAddress()
}
