package main

//
// Forward path router
//

import (
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.zx2c4.com/wireguard/tun"
)

// forwardpathRouter is the forward path router.
func forwardpathRouter(dnat *dnatState, conn net.Conn, devTUN tun.Device) {
	for {
		// step 1: read tunneled frame from miniooni from the conn device
		frame, err := netxlite.ReadSimpleFrame(conn)
		if err != nil {
			log.Warnf("forwardpath: ReadSimpleFrame: %s", err.Error())
			return
		}

		// step 2: parse packet as an IPv4 packet
		pkt := gopacket.NewPacket(frame, layers.LayerTypeIPv4, gopacket.Default)
		players := pkt.Layers()
		if len(players) < 2 {
			log.Warnf("fwdPath: drop packet: too few layers: %+v", frame)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("forwardpath: drop packet: not IPv4: %+v", frame)
			continue
		}

		// step 3: dispatch to UDP or TCP and otherwise drop
		udp, good := players[1].(*layers.UDP)
		if good {
			forwardpathRouteUDPv4(dnat, devTUN, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			forwardpathRouteTCPv4(dnat, devTUN, ipv4, tcp)
			continue
		}
		log.Warnf("forwardpath: drop packet: neither UDP nor TCP: %+v", frame)
	}
}

// forwardpathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.
func forwardpathRouteUDPv4(dnat *dnatState, devTUN tun.Device, ipv4 *layers.IPv4, udp *layers.UDP) {
	packet := dnat.rewriteForwardUDPv4(ipv4, udp)
	const zeroOffset = 0
	if _, err := devTUN.Write(packet, zeroOffset); err != nil {
		log.Warnf("forwardpathRouteUDPv4: Write: %s", err.Error())
	}
}

// forwardpathRouteTCPv4 routes an IPv4 packet containing a TCP segment.
func forwardpathRouteTCPv4(dnat *dnatState, devTUN tun.Device, ipv4 *layers.IPv4, tcp *layers.TCP) {
	packet := dnat.rewriteForwardTCPv4(ipv4, tcp)
	const zeroOffset = 0
	if _, err := devTUN.Write(packet, zeroOffset); err != nil {
		log.Warnf("forwardpathRouteTCPv4: Write: %s", err.Error())
	}
}
