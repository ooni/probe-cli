package main

//
// Forward path: from proxy to proxied services
//

import (
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.zx2c4.com/wireguard/tun"
)

// fwdPathLoop is the forward path loop.
func fwdPathLoop(devUDP net.PacketConn, devTUN tun.Device) {
	buffer := make([]byte, 4096)
	for {
		count, source, err := devUDP.ReadFrom(buffer)
		if err != nil {
			log.Warnf("fwdPath: ReadFrom: %s", err.Error())
			continue
		}
		payload := buffer[:count]
		pkt := gopacket.NewPacket(payload, layers.LayerTypeIPv4, gopacket.Default)
		players := pkt.Layers()
		if len(players) < 2 {
			log.Warnf("fwdPath: drop packet: too few layers: %+v", payload)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("fwdPath: drop packet: not IPv4: %+v", payload)
			continue
		}
		udp, good := players[1].(*layers.UDP)
		if good {
			fwdPathRouteUDPv4(devTUN, source, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			fwdPathRouteTCPv4(devTUN, source, ipv4, tcp)
			continue
		}
		log.Warnf("fwdPath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// fwdPathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.
func fwdPathRouteUDPv4(devTUN tun.Device, tunnelTo net.Addr, ipv4 *layers.IPv4, udp *layers.UDP) {
	rewritten := natRewriteForwardUDPv4(tunnelTo, ipv4, udp)
	const zeroOffset = 0
	if _, err := devTUN.Write(rewritten, zeroOffset); err != nil {
		log.Warnf("fwdPathRouteUDPv4: Write: %s", err.Error())
	}
}

// fwdPathRouteTCPv4 routes an IPv4 packet containing a TCP segment.
func fwdPathRouteTCPv4(devTUN tun.Device, tunnelTo net.Addr, ipv4 *layers.IPv4, udp *layers.TCP) {
	// nothing
}
