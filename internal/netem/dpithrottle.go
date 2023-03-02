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

	// DPIStack is the stack we wrap.
	DPIStack
}

var _ BackboneStack = &DPIThrottleTrafficForTLSSNI{}

// NewDPIThrottleTrafficForTLSSNI constructs a [DPIThrottleTrafficForTLSSNI].
func NewDPIThrottleTrafficForTLSSNI(
	stack DPIStack,
	sni string,
	targetPLR float64,
) *DPIThrottleTrafficForTLSSNI {
	return &DPIThrottleTrafficForTLSSNI{
		plrm:     newLinkLossesManager(targetPLR),
		slowed:   &dpiFlowList{},
		sni:      sni,
		DPIStack: stack,
	}
}

// ReadPacket implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) ReadPacket() ([]byte, error) {
	rawPacket, err := bs.DPIStack.ReadPacket()
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

	// if the packet is offending, register it
	if sni == bs.sni {
		bs.slowed.addFromPacket(packet)
	}

	// deliver packet ANYWAY
	return rawPacket, nil
}

// WritePacket implements BackboneStack
func (bs *DPIThrottleTrafficForTLSSNI) WritePacket(rawPacket []byte) error {
	// parse the packet
	packet, err := dissectPacket(rawPacket)
	if err != nil {
		return bs.DPIStack.WritePacket(rawPacket)
	}

	// if this packet is slowed down check whether we should drop it
	if bs.slowed.belongsTo(LinkDirectionRightToLeft, packet) {
		if bs.plrm.shouldDrop() {
			return nil
		}
		// fallthrough
	}

	// deliver the packet
	return bs.DPIStack.WritePacket(rawPacket)
}
