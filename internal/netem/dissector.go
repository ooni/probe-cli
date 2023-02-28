package netem

//
// Protocol dissector
//

import (
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// dissectedPacket is a dissected packet.
type dissectedPacket struct {
	// pkt is the underlying packet.
	pkt gopacket.Packet

	// ip is the network layer.
	ip gopacket.NetworkLayer

	// tcp is the POSSIBLY NIL tcp layer.
	tcp *layers.TCP

	// udp is the POSSIBLY NIL UDP layer.
	udp *layers.UDP
}

// errDissectShortPacket indicates the packet is too short.
var errDissectShortPacket = errors.New("dissect: packet too short")

// errDissectNetwork indicates that we don't support the packet's network protocol.
var errDissectNetwork = errors.New("dissect: unsupported network protocol")

// errDissectTransport indicates that we don't support the packet's transport protocol.
var errDissectTransport = errors.New("dissect: unsupported transport protocol")

// dissect parses a packet TCP/IP layers.
func dissect(rawPacket []byte) (*dissectedPacket, error) {
	dp := &dissectedPacket{}

	// we need to sniff the protocol version
	if len(rawPacket) < 1 {
		return nil, errDissectShortPacket
	}
	version := uint8(rawPacket[0]) >> 4

	// parse the IP layer
	switch {
	case version == 4:
		dp.pkt = gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Lazy)
		ipLayer := dp.pkt.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			return nil, errDissectNetwork
		}
		dp.ip = ipLayer.(*layers.IPv4)

	case version == 6:
		dp.pkt = gopacket.NewPacket(rawPacket, layers.LayerTypeIPv6, gopacket.Lazy)
		ipLayer := dp.pkt.Layer(layers.LayerTypeIPv6)
		if ipLayer == nil {
			return nil, errDissectNetwork
		}
		dp.ip = ipLayer.(*layers.IPv6)

	default:
		return nil, errDissectNetwork
	}

	// parse the transport layer
	switch dp.transportProtocol() {
	case layers.IPProtocolTCP:
		dp.tcp = dp.pkt.Layer(layers.LayerTypeTCP).(*layers.TCP)

	case layers.IPProtocolUDP:
		dp.udp = dp.pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)

	default:
		return nil, errDissectTransport
	}

	return dp, nil
}

// decrementTimeToLive decrements the IPv4 or IPv6 time to live.
func (dp *dissectedPacket) decrementTimeToLive() {
	switch v := dp.ip.(type) {
	case *layers.IPv4:
		v.TTL--
	case *layers.IPv6:
		v.HopLimit--
	default:
		panic(errDissectNetwork)
	}
}

// timeToLive returns the packet's IPv4 or IPv6 time to live.
func (dp *dissectedPacket) timeToLive() int64 {
	switch v := dp.ip.(type) {
	case *layers.IPv4:
		return int64(v.TTL)
	case *layers.IPv6:
		return int64(v.HopLimit)
	default:
		panic(errDissectNetwork)
	}
}

// destinationIPAddress returns the packet's destination IP address.
func (dp *dissectedPacket) destinationIPAddress() string {
	switch v := dp.ip.(type) {
	case *layers.IPv4:
		return v.DstIP.String()
	case *layers.IPv6:
		return v.DstIP.String()
	default:
		panic(errDissectNetwork)
	}
}

// sourceIPAddress returns the packet's source IP address.
func (dp *dissectedPacket) sourceIPAddress() string {
	switch v := dp.ip.(type) {
	case *layers.IPv4:
		return v.SrcIP.String()
	case *layers.IPv6:
		return v.SrcIP.String()
	default:
		panic(errDissectNetwork)
	}
}

// transportProtocol returns the packet's transport protocol.
func (dp *dissectedPacket) transportProtocol() layers.IPProtocol {
	switch v := dp.ip.(type) {
	case *layers.IPv4:
		return v.Protocol
	case *layers.IPv6:
		return v.NextHeader
	default:
		panic(errDissectNetwork)
	}
}

// serialize serializes a previously dissected and modified packet.
func (dp *dissectedPacket) serialize() ([]byte, error) {
	switch {
	case dp.tcp != nil:
		dp.tcp.SetNetworkLayerForChecksum(dp.ip)
	case dp.udp != nil:
		dp.udp.SetNetworkLayerForChecksum(dp.ip)
	default:
		return nil, errDissectTransport
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializePacket(buf, opts, dp.pkt); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// matchDestination returns true when the given IPv4 packet has the
// expected protocol, destination address, and port.
func (dp *dissectedPacket) matchDestination(proto layers.IPProtocol, address string, port uint16) bool {
	if dp.transportProtocol() != proto {
		return false
	}
	switch {
	case dp.tcp != nil:
		return dp.destinationIPAddress() == address && dp.tcp.DstPort == layers.TCPPort(port)
	case dp.udp != nil:
		return dp.destinationIPAddress() == address && dp.udp.DstPort == layers.UDPPort(port)
	default:
		return false
	}
}

// matchSource returns true when the given IPv4 packet has the
// expected protocol, source address, and port.
func (dp *dissectedPacket) matchSource(proto layers.IPProtocol, address string, port uint16) bool {
	if dp.transportProtocol() != proto {
		return false
	}
	switch {
	case dp.tcp != nil:
		return dp.sourceIPAddress() == address && dp.tcp.SrcPort == layers.TCPPort(port)
	case dp.udp != nil:
		return dp.sourceIPAddress() == address && dp.udp.SrcPort == layers.UDPPort(port)
	default:
		return false
	}
}
