package netem

//
// Composable deep-packet-inspection rules
//

import (
	"context"

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
	// Check whether packet is flowing in the expected direction.
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
