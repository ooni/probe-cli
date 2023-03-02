package netem

//
// DPI: rules to block flows
//

import (
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
)

// DPIResetTrafficForTLSSNI is a [LinkDPIEngine] that sends
// a RST TCP segment after it sees a given TLS SNI. The zero value is
// invalid; construct using [NewDPIResetTrafficForTLSSNI].
type DPIResetTrafficForTLSSNI struct {
	// sni is the offending SNI.
	sni string

	// DPIStack is the stack we wrap.
	DPIStack
}

var _ BackboneStack = &DPIResetTrafficForTLSSNI{}

// NewDPIResetTrafficForTLSSNI constructs a [DPIResetTrafficForTLSSNI].
func NewDPIResetTrafficForTLSSNI(stack DPIStack, sni string) *DPIResetTrafficForTLSSNI {
	return &DPIResetTrafficForTLSSNI{
		sni:      sni,
		DPIStack: stack,
	}
}

// ReadPacket implements BackboneStack
func (bs *DPIResetTrafficForTLSSNI) ReadPacket() ([]byte, error) {
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

	// if the packet is offending, send the RST soon
	if sni == bs.sni {
		rawResponse, err := reflectDissectedTCPSegmentWithRSTFlag(packet)
		if err != nil {
			log.Warnf("DPIResetTrafficForTLSSNI: %s", err.Error())
			return nil, err
		}
		<-time.After(2 * time.Millisecond)
		_ = bs.DPIStack.WritePacket(rawResponse)
	}

	// deliver packet ANYWAY
	return rawPacket, nil
}

// WritePacket implements BackboneStack
func (bs *DPIResetTrafficForTLSSNI) WritePacket(rawPacket []byte) error {
	return bs.DPIStack.WritePacket(rawPacket)
}
