package main

//
// DNAT implementation
//
// The IP address assigned to miniooni is always 10.17.17.4. The services
// running inside this program always use 10.17.17.1. The job of this DNAT
// code is to map any destination address emitted by the probe to 10.17.17.1
// on the forward path (probe->services) and to map back 10.17.17.1 to the
// original IP address used by miniooni in the return path.
//
// The forward path portion of this code upserts into the DNAT table an
// entry containing the real destination address. We use as key:
//
// 1. protocol (TCP/UDP);
//
// 2. source port;
//
// 3. destination port.
//
// After that, we rewrite the destination addess to be 10.17.17.1.
//
// The return path portion of this code retrieves the original destination
// address using as key (note that the order matters!):
//
// 1. protocol (TCP/UDP);
//
// 2. destination port;
//
// 3. source port.
//
// A general implementation of the QUIC protocol breaks this design
// because it uses the same local socket to communicate to several
// QUIC servers concurrently. However, the OONI implementation uses
// a new UDP socket for each new destination server, so we should
// actually be fine. If/when this fact is not true anymore, we need
// to write special purpose code to use the QUIC connection ID for
// routing instead of a subset of the five tuple.
//

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// dnatState contains the DNAT state.
type dnatState struct {
	// mu provides mutual exclusion
	mu sync.Mutex

	// origDstIP maps to the original destination IP
	origDstIP map[string]net.IP

	// tcp contains the state for tcp
	tcp map[string]*dnatRecord

	// udp contains the state for udp
	udp map[string]*dnatRecord
}

// dnatRecord is a DNAT record.
type dnatRecord struct {
	// origDstIP is the original source IP address
	origDstIP net.IP
}

// upsertRecord creates or updates state.
func (ds *dnatState) upsertRecord(
	protocol uint8,
	srcIP net.IP,
	srcPort uint16,
	dstIP net.IP,
	dstPort uint16,
) {
	ds.mu.Lock()
	rec := &dnatRecord{
		origDstIP: dstIP,
	}
	key := fmt.Sprintf("%d_%d", srcPort, dstPort)
	switch protocol {
	case uint8(layers.IPProtocolTCP):
		ds.tcp[key] = rec
	case uint8(layers.IPProtocolUDP):
		ds.udp[key] = rec
	}
	ds.mu.Unlock()
}

// errDNATNoSuchRecord indicates there's no DNAT record.
var errDNATNoSuchRecord = errors.New("dnat: no such record")

// getRecord obtains the record for a given five tuple.
func (ds *dnatState) getRecord(
	protocol uint8,
	srcIP net.IP,
	srcPort uint16,
	dstIP net.IP,
	dstPort uint16,
) (*dnatRecord, error) {
	defer ds.mu.Unlock()
	ds.mu.Lock()
	var rec *dnatRecord
	key := fmt.Sprintf("%d_%d", dstPort, srcPort) // swapped: this is the return path!
	switch protocol {
	case uint8(layers.IPProtocolTCP):
		rec = ds.tcp[key]
	case uint8(layers.IPProtocolUDP):
		rec = ds.udp[key]
	}
	if rec == nil {
		return nil, errDNATNoSuchRecord
	}
	return rec, nil
}

// rewriteForwardUDPv4 attempts to rewrite an UDPv4 packet on the forward path
func (ds *dnatState) rewriteForwardUDPv4(ipv4 *layers.IPv4, udp *layers.UDP) []byte {
	// step 1: upsert into the NAT table
	ds.upsertRecord(
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

// rewriteReturnUDPv4 attempts to rewrite an UDPv4 packet on the return path
func (ds *dnatState) rewriteReturnUDPv4(ipv4 *layers.IPv4, udp *layers.UDP) ([]byte, error) {
	// step 1: access the NAT table
	rec, err := ds.getRecord(
		uint8(ipv4.Protocol),
		ipv4.SrcIP,
		uint16(udp.SrcPort),
		ipv4.DstIP,
		uint16(udp.DstPort),
	)
	if err != nil {
		return nil, err
	}

	// step 2: rewrite the source IP address
	ipv4.SrcIP = rec.origDstIP

	// step 3: serialize the modified packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	udp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err = gopacket.SerializeLayers(buf, opts, ipv4, udp, gopacket.Payload(udp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	return buf.Bytes(), nil
}

// rewriteForwardTCPv4 attempts to rewrite an TCPv4 packet on the forward path
func (ds *dnatState) rewriteForwardTCPv4(ipv4 *layers.IPv4, tcp *layers.TCP) []byte {
	// step 1: upsert into the NAT table
	ds.upsertRecord(
		uint8(ipv4.Protocol),
		ipv4.SrcIP,
		uint16(tcp.SrcPort),
		ipv4.DstIP,
		uint16(tcp.DstPort),
	)

	// step 2: rewrite the destination IP address
	ipv4.DstIP = net.IPv4(10, 17, 17, 1)

	// step 3: serialize the modified packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	tcp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err := gopacket.SerializeLayers(buf, opts, ipv4, tcp, gopacket.Payload(tcp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	return buf.Bytes()
}

// rewriteReturnTCPv4 attempts to rewrite an TCPPv4 packet on the return path
func (ds *dnatState) rewriteReturnTCPv4(ipv4 *layers.IPv4, tcp *layers.TCP) ([]byte, error) {
	// step 1: access the NAT table
	rec, err := ds.getRecord(
		uint8(ipv4.Protocol),
		ipv4.SrcIP,
		uint16(tcp.SrcPort),
		ipv4.DstIP,
		uint16(tcp.DstPort),
	)
	if err != nil {
		return nil, err
	}

	// step 2: rewrite the source IP address
	ipv4.SrcIP = rec.origDstIP

	// step 3: serialize the modified packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	tcp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err = gopacket.SerializeLayers(buf, opts, ipv4, tcp, gopacket.Payload(tcp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	return buf.Bytes(), nil
}
