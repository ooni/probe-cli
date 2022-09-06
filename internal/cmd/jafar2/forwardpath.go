package main

//
// Forward path router
//

import (
	"net"
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
)

// forwardPathRouter is the forward path router.
//
// Arguments:
//
// - dnatState keeps the DNAT state;
//
// - miniooniConn is the connection with the miniooni client;
//
// - devTCP is the device where to write TCP segments;
//
// - devUDP is the packet conn to use to send UDP datagrams.
func forwardPathRouter(
	dnat *dnatState,
	miniooniConn net.Conn,
	devTCP tun.Device,
	devUDP net.PacketConn,
) {
	for {
		// step 1: read tunneled frame from miniooni from the conn device
		frame, err := netxlite.ReadSimpleFrame(miniooniConn)
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
			forwardPathRouteUDPv4(dnat, devUDP, ipv4, udp)
			continue
		}
		tcp, good := players[1].(*layers.TCP)
		if good {
			forwardPathRouteTCPv4(dnat, devTCP, ipv4, tcp)
			continue
		}
		log.Warnf("forwardpath: drop packet: neither UDP nor TCP: %+v", frame)
	}
}

// forwardPathRouteUDPv4 routes an IPv4 packet containing an UDP datagram.
func forwardPathRouteUDPv4(
	dnat *dnatState,
	miniooniConn net.Conn,
	devUDP net.PacketConn,
	ipv4 *layers.IPv4,
	udp *layers.UDP,
) {
	// assemble the destination address from available data
	destAddr := &net.UDPAddr{
		IP:   ipv4.DstIP,
		Port: int(udp.DstPort),
		Zone: "",
	}

	// create socket for sending this datagram
	pconn, err := net.Dial("udp", destAddr.String())
	runtimex.PanicOnError(err, "net.ListenUDP")
	defer pconn.Close()

	// set reasonable timeout for this UDP socket
	const timeout = 4 * time.Second
	pconn.SetDeadline(time.Now().Add(timeout))

	// send payload to the server
	if _, err := pconn.Write(udp.Payload); err != nil {
		log.Warnf("forwardpathRouteUDPv4: Write: %s", err.Error())
		return
	}

	// receive response
	buffer := make([]byte, 4096)
	count, err := pconn.Read(buffer)
	if err != nil {
		log.Warnf("forwardpathRouteUDPv4: Read: %s", err.Error())
		return
	}
	payload := buffer[:count]

	// prepare for reflecting the original datagram back
	ipv4.SrcIP, ipv4.DstIP = ipv4.DstIP, ipv4.SrcIP
	udp.SrcPort, udp.DstPort = udp.DstPort, udp.SrcPort

	// serialize to packet buffer
	packetBuffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	udp.SetNetworkLayerForChecksum(ipv4) // see https://github.com/google/gopacket/issues/290
	err = gopacket.SerializeLayers(packetBuffer, opts, ipv4, udp, gopacket.Payload(payload))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")

	// send packet back to miniooni
	rawPacket := packetBuffer.Bytes()
	if err := netxlite.WriteSimpleFrame(miniooniConn, rawPacket); err != nil {
		log.Warnf("returnpathUDPRouter: netxlite.WriteSimpleFrame: %s", err.Error())
		return
	}
}

// forwardPathRouteTCPv4 routes an IPv4 packet containing a TCP segment.
//
// The strategy for TCP consists of registering an entry inside the DNAT table,
// rewriting the destination address, and then sending over to [devTCP]. This kind
// of send passes the bytes to an application-level instance of TCP, which will
// reassemble the data and then send upstream to the real server.
func forwardPathRouteTCPv4(dnat *dnatState, devTCP tun.Device, ipv4 *layers.IPv4, tcp *layers.TCP) {
	// TODO(bassosimone): we should refactor this algorithm to be more explicit
	packet := dnat.rewriteForwardTCPv4(ipv4, tcp)
	const zeroOffset = 0
	if _, err := devTCP.Write(packet, zeroOffset); err != nil {
		log.Warnf("forwardpathRouteTCPv4: Write: %s", err.Error())
	}
}
