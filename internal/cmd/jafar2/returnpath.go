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

// returnPathLoop is the return path loop.
func returnPathLoop(devTUN tun.Device, devUDP net.PacketConn) {
	const zeroOffset = 0
	buffer := make([]byte, 4096)
	for {
		count, err := devTUN.Read(buffer, zeroOffset)
		if err != nil {
			log.Warnf("returnPath: Read: %s", err.Error())
			continue
		}
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
		udp, good := players[1].(*layers.UDP)
		if good {
			returnPathRouteUDPv4(devUDP, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			returnPathRouteTCPv4(devUDP, ipv4, tcp)
			continue
		}
		log.Warnf("returnPath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// returnPathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.
func returnPathRouteUDPv4(devUDP net.PacketConn, ipv4 *layers.IPv4, udp *layers.UDP) {
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
func returnPathRouteTCPv4(devUDP net.PacketConn, ipv4 *layers.IPv4, udp *layers.TCP) {
	// nothing
}
