package main

import (
	"errors"
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

	// state contains the actual state
	state map[uint16]*dnatRecord
}

// dnatRecord is a DNAT record.
type dnatRecord struct {
	// origDstIP is the original source IP address
	origDstIP net.IP
}

// upsertRecord creates or updates state.
func (ds *dnatState) upsertRecord(
	protocol uint8, srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16) {
	ds.mu.Lock()
	ds.state[uint16(srcPort)] = &dnatRecord{
		origDstIP: dstIP,
	}
	ds.mu.Unlock()
}

// errDNATNoSuchRecord indicates there's no DNAT record.
var errDNATNoSuchRecord = errors.New("dnat: no such record")

// getRecord obtains the record for a given five tuple.
func (ds *dnatState) getRecord(protocol uint8, srcIP net.IP, srcPort uint16,
	dstIP net.IP, dstPort uint16) (*dnatRecord, error) {
	defer ds.mu.Unlock()
	ds.mu.Lock()
	record := ds.state[uint16(dstPort)] // we're on the return path so use the dstPort
	if record == nil {
		return nil, errDNATNoSuchRecord
	}
	return record, nil
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
