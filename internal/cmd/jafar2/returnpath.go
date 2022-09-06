package main

//
// Return path router
//

import (
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.zx2c4.com/wireguard/tun"
)

// returnpathRouter is the return path router.
func returnpathRouter(dnat *dnatState, devTUN tun.Device, conn net.Conn) {
	const zeroOffset = 0
	buffer := make([]byte, 4096)

	for {
		// step 1: read tunneled packet on [devTUN] sent by proxies
		count, err := devTUN.Read(buffer, zeroOffset)
		if err != nil {
			log.Warnf("returnpath: Read: %s", err.Error())
			return
		}

		// step 2: parse packet as an IPv4 packet
		payload := buffer[:count]
		pkt := gopacket.NewPacket(payload, layers.LayerTypeIPv4, gopacket.Default)
		players := pkt.Layers()
		if len(players) < 2 {
			log.Warnf("returnpath: drop packet: too few layers: %+v", payload)
			continue
		}
		ipv4, good := players[0].(*layers.IPv4)
		if !good {
			log.Warnf("returnpath: drop packet: not IPv4: %+v", payload)
			continue
		}

		// step 3: dispatch to UDP or TCP and otherwise drop
		udp, good := players[1].(*layers.UDP)
		if good {
			returnpathRouteUDPv4(dnat, conn, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			returnpathRouteTCPv4(dnat, conn, ipv4, tcp)
			continue
		}
		log.Warnf("returnpath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// returnpathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.
func returnpathRouteUDPv4(dnat *dnatState, conn net.Conn, ipv4 *layers.IPv4, udp *layers.UDP) {
	frame, err := dnat.rewriteReturnUDPv4(ipv4, udp)
	if err != nil {
		log.Warnf("returnPathRouteUDPv4: natRewriteReturnUDPv4: %s", err.Error())
		return
	}
	if err := netxlite.WriteSimpleFrame(conn, frame); err != nil {
		log.Warnf("returnPathRouteUDPv4: Write: %s", err.Error())
	}
}

// returnpathRouteTCPv4 routes an IPv4 packet containing a TCP segment.
func returnpathRouteTCPv4(dnat *dnatState, conn net.Conn, ipv4 *layers.IPv4, tcp *layers.TCP) {
	frame, err := dnat.rewriteReturnTCPv4(ipv4, tcp)
	if err != nil {
		log.Warnf("returnPathRouteTCPv4: natRewriteReturnTCPv4: %s", err.Error())
		return
	}
	if err := netxlite.WriteSimpleFrame(conn, frame); err != nil {
		log.Warnf("returnPathRouteTCPv4: Write: %s", err.Error())
	}
}
