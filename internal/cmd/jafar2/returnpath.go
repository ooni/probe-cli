package main

//
// Return path (client<-proxies)
//
// By reading this file top-down you get a sense of the travel
// performed by packets from proxies to client.
//

import (
	"context"
	"net"
	"sync"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
)

// tcpDeviceReader reads and processes packets sent by the proxies.
//
// Arguments:
//
// - ctx is the context binding the lifetime of this goroutine;
//
// - wg is the wait group used by the parent;
//
// - tcpState is the TCP state for implementing DNAT;
//
// - packetsForClient is the queue where to append packets for the client;
//
// - tcpDev is the virtual device connected to TCP servers.
//
// This function runs in a goroutine that keeps running until [ctx] is
// not done and [clientConn] does not emit errors.
func tcpDeviceReader(
	ctx context.Context,
	wg *sync.WaitGroup,
	tcpState *tcpState,
	tcpDev tun.Device,
	packetsForClient chan<- []byte,
) {
	// notify termination
	defer wg.Done()

	buffer := make([]byte, 4096)
	for ctx.Err() == nil {

		// read the new incoming raw packet
		const zeroOffset = 0
		count, err := tcpDev.Read(buffer, zeroOffset)
		if err != nil {
			log.Warnf("tcpDeviceReader: tcpDev.Read: %s", err.Error())
			return
		}
		rawPacket := buffer[:count]

		// check whether packet is IPv4 and otherwise discard it
		packet := gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Default)
		players := packet.Layers()
		if len(players) < 2 {
			log.Warnf("tcpDeviceReader: drop packet: too few layers: %+v", rawPacket)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("tcpDeviceReader: drop packet: not IPv4: %+v", rawPacket)
			continue
		}

		// process incoming TCP packets
		if tcp, good := players[1].(*layers.TCP); good {
			tcpDevReaderTCPv4(ctx, tcpState, packetsForClient, ipv4, tcp)
			continue
		}

		// discard all the other incoming packets
		log.Warnf("tcpDeviceReader: drop packet: neither UDP nor TCP: %+v", rawPacket)
	}
}

// tcpDevReaderTCPv4 processes a TCPv4 datagram from the proxies.
func tcpDevReaderTCPv4(
	ctx context.Context,
	state *tcpState,
	packetsForClient chan<- []byte,
	ipv4 *layers.IPv4,
	tcp *layers.TCP,
) {
	// Undo the effects of DNAT
	var srcIP net.IP
	state.mu.Lock()
	srcIP = state.dnat[uint16(tcp.DstPort)]
	state.mu.Unlock()
	if srcIP == nil {
		log.Warnf("tcpDevReaderTCPv4: missing DNAT entry for %d", tcp.DstPort)
		return
	}
	ipv4.SrcIP = srcIP

	// serialize to bytes
	pktbuf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	tcp.SetNetworkLayerForChecksum(ipv4)
	err := gopacket.SerializeLayers(pktbuf, opts, ipv4, tcp, gopacket.Payload(tcp.Payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")
	rawPacket := pktbuf.Bytes()

	// forward packet to the client queue
	select {
	case packetsForClient <- rawPacket:
	case <-ctx.Done():
	}
}

// clientConnWriter reads the queue of packets sent to the client
// and emits each of them in sequence. This function runs in a goroutine
// that keeps running until [ctx] is not done and [clientConn] is OK.
func clientConnWriter(
	ctx context.Context,
	wg *sync.WaitGroup,
	rawPackets <-chan []byte,
	clientConn net.Conn,
) {
	// notify termination
	defer wg.Done()

	for {
		select {

		// We have a ready to send raw packet
		case rawPacket := <-rawPackets:
			if err := netxlite.WriteSimpleFrame(clientConn, rawPacket); err != nil {
				log.Warnf("clientConnWriter: netxlite.WriteSimpleFrame: %s", err.Error())
				return
			}

		// We were asked to terminate
		case <-ctx.Done():
			return
		}
	}
}
