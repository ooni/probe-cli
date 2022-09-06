package main

//
// Return path router
//

import (
	"fmt"
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
)

// returnPathTCPRouter is the return path router.
//
// Arguments:
//
// - dnatState keeps the DNAT state;
//
// - miniooniConn is the connection with the miniooni client;
//
// - devTCP is the device where to write TCP segments.
func returnPathTCPRouter(dnat *dnatState, miniooniConn net.Conn, devTCP tun.Device) {
	const zeroOffset = 0
	buffer := make([]byte, 4096)

	for {
		// step 1: read tunneled packet on [devTCP] sent by proxies
		count, err := devTCP.Read(buffer, zeroOffset)
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

		// step 3: dispatch to TCP and otherwise drop
		tcp, good := players[1].(*layers.TCP)
		if good {
			returnpathRouteTCPv4(dnat, miniooniConn, ipv4, tcp)
			continue
		}
		log.Warnf("returnpath: drop packet: neither UDP nor TCP: %+v", payload)
	}
}

// returnpathTCPRouter is the return path router.
//
// Arguments:
//
// - dnatState keeps the DNAT state;
//
// - miniooniConn is the connection with the miniooni client;
//
// - devUDP is the device from which to read UDP packets.
func returnPathUDPRouter(dnat *dnatState, miniooniConn net.Conn, devUDP net.PacketConn) {
	buffer := make([]byte, 4096)
	dstIP, dstPort, err := twoTuple(devUDP.LocalAddr())
	runtimex.PanicOnError(err, "twoTuple")
	for {

		// read payload from remote server
		count, source, err := devUDP.ReadFrom(buffer)
		if err != nil {
			log.Warnf("returnpathUDPRouter: ReadFrom: %s", err.Error())
			return
		}
		payload := buffer[:count]

		// obtain the port from which the server contacted us
		_, srcPort, err := twoTuple(source)
		runtimex.PanicOnError(err, "twoTuple")

		// obtain the original source address
		var srcIP net.IP
		keyDNAT := fmt.Sprintf("udp_%s_%s")
		dnat.mu.Lock()
		srcIP = dnat.origDstIP[keyDNAT]
		dnat.mu.Unlock()

		rec, err := dnat.getRecord(
			uint8(layers.IPProtocolUDP), // UDP
			srcIP,
			srcPort,
			dstIP,
			dstPort,
		)
		if err != nil {
			log.Warnf("returnpathUDPRouter: dnat.getRecord: %s", err.Error())
			continue
		}
		udp := &layers.UDP{
			BaseLayer: layers.BaseLayer{},
			SrcPort:   layers.UDPPort(srcPort),
			DstPort:   layers.UDPPort(dstPort),
			Length:    0,
			Checksum:  0,
		}
		ipv4 := &layers.IPv4{
			BaseLayer:  layers.BaseLayer{},
			Version:    4,
			IHL:        0,
			TOS:        0,
			Length:     0,
			Id:         0, // TODO(bassosimone)
			Flags:      0,
			FragOffset: 0,
			TTL:        14,
			Protocol:   layers.IPProtocolUDP,
			Checksum:   0,
			SrcIP:      rec.origDstIP,
			DstIP:      net.IPv4(10, 17, 17, 1),
			Options:    []layers.IPv4Option{},
			Padding:    []byte{},
		}
		// step 3: serialize the modified packet
		packetBuffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}
		udp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
		err = gopacket.SerializeLayers(packetBuffer, opts, ipv4, udp, gopacket.Payload(payload))
		runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

		// step 4: send packet to miniooni
		rawPacket := packetBuffer.Bytes()
		if err := netxlite.WriteSimpleFrame(miniooniConn, rawPacket); err != nil {
			log.Warnf("returnpathUDPRouter: netxlite.WriteSimpleFrame: %s", err.Error())
			return
		}
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
	// TODO(bassosimone): we should refactor this algorithm to be more explicit
	frame, err := dnat.rewriteReturnTCPv4(ipv4, tcp)
	if err != nil {
		log.Warnf("returnPathRouteTCPv4: natRewriteReturnTCPv4: %s", err.Error())
		return
	}
	if err := netxlite.WriteSimpleFrame(conn, frame); err != nil {
		log.Warnf("returnPathRouteTCPv4: Write: %s", err.Error())
	}
}
