package main

//
// Forward path: from proxy to proxied services
//

import (
	"errors"
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.zx2c4.com/wireguard/tun"
)

// fwdPathLoop is the forward path loop. This is the algorithm:
//
// 1. read a raw packet from the UDP tunnel [devUDP];
//
// 2. exclude packets that do not parse as IPv4;
//
// 3. process TCP or UDP packets and drop all the other packets.
//
// This function loops until [devUDP] is closed.
func fwdPathLoop(devUDP net.PacketConn, devTUN tun.Device) {
	buffer := make([]byte, 4096)

	for {
		// step 1: read tunneled packet from miniooni from the devUDP device
		count, source, err := devUDP.ReadFrom(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return // as documented
			}
			log.Warnf("fwdPath: ReadFrom: %s", err.Error())
			continue
		}

		// step 2: parse packet as an IPv4 packet
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

		// step 3: dispatch to UDP or TCP and otherwise drop
		udp, good := players[1].(*layers.UDP)
		if good {
			fwdPathRouteUDPv4(devTUN, payload, source, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			fwdPathRouteTCPv4(devTUN, payload, source, ipv4, tcp)
			continue
		}
		log.Warnf("fwdPath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// fwdPathRouteUDPv4 routes an IPv4 packet containing an UDP datagram. The algorithm
// implemented by this function is the following:
//
// 1. check whether we should drop this packet;
//
// 2. otherwise, rewrite the four tuple using the NAT;
//
// 3. forward to the [devTUN] device.
//
// This function never fails and just logs errors that may occur.
func fwdPathRouteUDPv4(
	devTUN tun.Device, payload []byte, tunnelTo net.Addr, ipv4 *layers.IPv4, udp *layers.UDP) {
	if filterShouldDropForwardUDPv4(ipv4, udp) {
		log.Warnf("fwdPathRouteUDPv4: drop packet: blocked by filter: %+v", payload)
		return
	}
	rawpkt := natRewriteForwardUDPv4(tunnelTo, ipv4, udp)
	const zeroOffset = 0
	if _, err := devTUN.Write(rawpkt, zeroOffset); err != nil {
		log.Warnf("fwdPathRouteUDPv4: Write: %s", err.Error())
	}
}

// fwdPathRouteTCPv4 routes an IPv4 packet containing a TCP segment. The implemented
// algorithm is similar to fwdPathRouteUDPv4's one but operates on TCP.
func fwdPathRouteTCPv4(
	devTUN tun.Device, payload []byte, tunnelTo net.Addr, ipv4 *layers.IPv4, tcp *layers.TCP) {
	if filterShouldDropForwardTCPv4(ipv4, tcp) {
		log.Warnf("fwdPathRouteTCPv4: drop packet: blocked by filter: %+v", payload)
		return
	}
	rawpkt := natRewriteForwardTCPv4(tunnelTo, ipv4, tcp)
	const zeroOffset = 0
	if _, err := devTUN.Write(rawpkt, zeroOffset); err != nil {
		log.Warnf("fwdPathRouteTCPv4: Write: %s", err.Error())
	}
}
