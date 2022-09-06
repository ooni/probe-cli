package main

//
// NAT implementation
//
// For now, this implementation does not change the source port and just
// rewrites destination IP address to divert to a local service. This design
// is of course simplified and only support basic scenarios.
//

import (
	"errors"
	"net"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	// natMu protects natTable
	natMu sync.Mutex

	// natTable contains the table used for NAT
	natTable = make(map[uint16]*natRecord)
)

// natRecord is a record used by NAT.
type natRecord struct {
	// tunnelTo is the address to tunnel the response to.
	tunnelTo net.Addr

	// protocol is the IP protocol.
	protocol uint8

	// srcIP is the original source IP address.
	srcIP net.IP

	// srcPort is the original source port.
	srcPort uint16

	// destIP is the original destination IP address.
	dstIP net.IP

	// dstPort is the original destination port.
	dstPort uint16
}

// natUpsertRecord creates or updates the NAT table.
func natUpsertRecord(tunnelTo net.Addr, protocol uint8,
	srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16) {
	natMu.Lock()
	natTable[srcPort] = &natRecord{
		tunnelTo: tunnelTo,
		protocol: protocol,
		srcIP:    srcIP,
		srcPort:  srcPort,
		dstIP:    dstIP,
		dstPort:  dstPort,
	}
	natMu.Unlock()
}

// natAccessRecord obtains the record for a given entry.
func natAccessRecord(protocol uint8, srcIP net.IP, srcPort uint16,
	dstIP net.IP, dstPort uint16) (*natRecord, bool) {
	defer natMu.Unlock()
	natMu.Lock()
	rec := natTable[dstPort]
	return rec, rec != nil
}

// natRewriteForwardUDPv4 rewrites an UDPv4 packet in the forward direction
func natRewriteForwardUDPv4(tunnelTo net.Addr, ipv4 *layers.IPv4, udp *layers.UDP) []byte {
	// step 1: upsert into the NAT table
	natUpsertRecord(
		tunnelTo,
		uint8(ipv4.Protocol),
		ipv4.SrcIP,
		uint16(udp.SrcPort),
		ipv4.DstIP,
		uint16(udp.DstPort),
	)

	// step 2: rewrite the destination IP address
	ipv4.DstIP = net.IPv4(10, 17, 17, 1)

	// step 3: serialize the modified packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	udp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err := gopacket.SerializeLayers(buf, opts, ipv4, udp, gopacket.Payload(udp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	return buf.Bytes()
}

// natRewriteReturnUDPv4 rewrites an UDPv4 packet in the return direction
// and returns the rewritten packet as well as the tunnelTo address.
func natRewriteReturnUDPv4(ipv4 *layers.IPv4, udp *layers.UDP) ([]byte, net.Addr, error) {
	// step 1: access the NAT table
	rec, found := natAccessRecord(
		uint8(ipv4.Protocol),
		ipv4.SrcIP,
		uint16(udp.SrcPort),
		ipv4.DstIP,
		uint16(udp.DstPort),
	)
	if !found {
		return nil, nil, errors.New("nat: no record")
	}

	// step 2: rewrite the source IP address
	ipv4.SrcIP = rec.dstIP

	// step 3: serialize the modified packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	udp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err := gopacket.SerializeLayers(buf, opts, ipv4, udp, gopacket.Payload(udp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	return buf.Bytes(), rec.tunnelTo, nil
}
