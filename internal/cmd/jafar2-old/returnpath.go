package main

//
// Return path: from proxied services to probe
//

import (
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.zx2c4.com/wireguard/tun"
)

// returnPathLoop is the return path loop. This is the algorithm:
//
// 1. read a raw packet from the UDP tunnel [devTUN];
//
// 2. exclude packets that do not parse as IPv4;
//
// 3. process TCP or UDP packets and drop all the other packets.
//
// This function loops until [devUDP] is closed.
func returnPathLoop(devTUN tun.Device, devUDP net.PacketConn) {
	const zeroOffset = 0
	buffer := make([]byte, 4096)

	for {
		// step 1: read tunneled packet on [devTUN] sent by proxies
		count, err := devTUN.Read(buffer, zeroOffset)
		if err != nil {
			log.Warnf("returnPath: Read: %s", err.Error())
			continue
		}

		// step 2: parse packet as an IPv4 packet
		payload := buffer[:count]
		pkt := gopacket.NewPacket(payload, layers.LayerTypeIPv4, gopacket.Default)
		players := pkt.Layers()
		if len(players) < 2 {
			log.Warnf("returnPath: drop packet: too few layers: %+v", payload)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("returnPath: drop packet: not IPv4: %+v", payload)
			continue
		}

		// step 3: dispatch to UDP or TCP and otherwise drop
		udp, good := players[1].(*layers.UDP)
		if good {
			returnPathRouteUDPv4(devUDP, payload, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			returnPathRouteTCPv4(devUDP, payload, ipv4, tcp)
			continue
		}
		log.Warnf("returnPath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// returnPathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.  The algorithm
// implemented by this function is the following:
//
// 1. check whether we should drop this packet;
//
// 2. otherwise, rewrite the four tuple using the NAT;
//
// 3. forward to the [devTUN] device.
//
// This function never fails and just logs errors that may occur.
func returnPathRouteUDPv4(devUDP net.PacketConn, payload []byte, ipv4 *layers.IPv4, udp *layers.UDP) {
	if filterShouldDropReturnUDPv4(ipv4, udp) {
		log.Warnf("returnPathRouteUDPv4: drop packet: blocked by filter: %+v", payload)
		return
	}
	rewritten, tunnelTo, err := natRewriteReturnUDPv4(ipv4, udp)
	if err != nil {
		log.Warnf("returnPathRouteUDPv4: natRewriteReturnUDPv4: %s", err.Error())
		return
	}
	if _, err := devUDP.WriteTo(rewritten, tunnelTo); err != nil {
		log.Warnf("returnPathRouteUDPv4: Write: %s", err.Error())
	}
}

// returnPathRouteTCPv4 routes an IPv4 packet containing a TCP segment.
func returnPathRouteTCPv4(devUDP net.PacketConn, payload []byte, ipv4 *layers.IPv4, tcp *layers.TCP) {
	if filterShouldDropReturnTCPv4(ipv4, tcp) {
		log.Warnf("returnPathRouteTCPv4: drop packet: blocked by filter: %+v", payload)
		return
	}
	rewritten, tunnelTo, err := natRewriteReturnTCPv4(ipv4, tcp)
	if err != nil {
		log.Warnf("returnPathRouteTCPv4: natRewriteReturnTCPv4: %s", err.Error())
		return
	}
	if _, err := devUDP.WriteTo(rewritten, tunnelTo); err != nil {
		log.Warnf("returnPathRouteTCPv4: Write: %s", err.Error())
	}
}
