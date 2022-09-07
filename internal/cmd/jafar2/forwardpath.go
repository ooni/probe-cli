package main

//
// Forward path (client->proxies)
//
// By reading this file top-down you get a sense of the travel
// performed by packets from client to proxies.
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

// clientConnReader reads and processes packets sent by the client.
//
// Arguments:
//
// - ctx is the context binding the lifetime of this goroutine;
//
// - wg is the wait group used by the parent;
//
// - clientConn is the TCP conn with the client (usually miniooni);
//
// - tcpState is the TCP state for implementing DNAT;
//
// - packetsForClient is the queue where to append packets for the client;
//
// - tcpDev is the virtual device connected to TCP servers.
//
// This function runs in a goroutine that keeps running until [ctx] is
// not done and [clientConn] does not emit errors.
func clientConnReader(
	ctx context.Context,
	wg *sync.WaitGroup,
	clientConn net.Conn,
	tcpState *tcpState,
	packetsForClient chan<- []byte,
	tcpDev tun.Device,
) {
	// notify termination
	defer wg.Done()

	for ctx.Err() == nil {
		// read raw frame from the client
		rawPacket, err := netxlite.ReadSimpleFrame(clientConn)
		if err != nil {
			log.Warnf("clientConnReader: netxlite.ReadSimpleFrame: %s", err.Error())
			return
		}

		// check whether packet is IPv4 and otherwise discard it
		packet := gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Default)
		players := packet.Layers()
		if len(players) < 2 {
			log.Warnf("clientConnReader: drop packet: too few layers: %+v", rawPacket)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("clientConnReader: drop packet: not IPv4: %+v", rawPacket)
			continue
		}

		// process incoming UDP packets
		if udp, good := players[1].(*layers.UDP); good {
			clientConnReaderUDPv4(ctx, rawPacket, ipv4, udp, packetsForClient)
			continue
		}

		// process incoming TCP packets
		if tcp, good := players[1].(*layers.TCP); good {
			clientConnReaderTCPv4(tcpState, tcpDev, ipv4, tcp)
			continue
		}

		// discard all the other incoming packets
		log.Warnf("clientConnReader: drop packet: neither UDP nor TCP: %+v", rawPacket)
	}
}

// clientConnReaderUDPv4 processes an UDPv4 datagram from the client.
func clientConnReaderUDPv4(
	ctx context.Context,
	rawPacket []byte,
	ipv4 *layers.IPv4,
	udp *layers.UDP,
	packetsForClient chan<- []byte,
) {
	switch udp.DstPort {
	case 53:
		dnsOverUDPv4(ctx, rawPacket, ipv4, udp, packetsForClient)
	default:
		log.Warnf("clientConnReaderUDPv4: drop packet: %+v", rawPacket)
	}
}

// clientConnReaderTCPv4 processes a TCPv4 datagram from the client.
func clientConnReaderTCPv4(
	state *tcpState,
	tcpDev tun.Device,
	ipv4 *layers.IPv4,
	tcp *layers.TCP,
) {
	// DNAT to the TCP proxyies attached to tcpDev
	state.mu.Lock()
	state.dnat[uint16(tcp.SrcPort)] = ipv4.DstIP
	state.mu.Unlock()
	ipv4.DstIP = net.IPv4(10, 17, 17, 1)

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

	// forward packet to the TCP proxies services
	const zeroOffset = 0
	if _, err := tcpDev.Write(rawPacket, zeroOffset); err != nil {
		log.Warnf("clientConnReaderTCPv4: dev.Write: %s", err.Error())
		return
	}
}
