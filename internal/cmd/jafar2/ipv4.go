package main

import (
	"context"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ipv4Forwarder reads from [rawch] and forwards the packet to either [tcpch] or [udpch] depending
// on the packet type. This goroutine returns when [ctx] is done. This function works for both
// the forward path (probe->internet) and the return path (internet->probe).
func ipv4Forwarder(
	ctx context.Context,
	rawch <-chan []byte,
	tcpch chan<- *tcpSegment,
	udpch chan<- *udpDatagram,
) {
	for {
		select {
		case rawPacket := <-rawch:
			packet := gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Default)
			players := packet.Layers()
			if len(players) < 2 {
				log.Warnf("fwdPath: drop packet: too few layers: %+v", rawPacket)
				continue
			}
			ipv4, good := players[0].(*layers.IPv4)
			if !good {
				log.Warnf("forwardpath: drop packet: not IPv4: %+v", rawPacket)
				continue
			}
			udp, good := players[1].(*layers.UDP)
			if good {
				datagram := &udpDatagram{
					packet: packet,
					ipv4:   ipv4,
					udp:    udp,
				}
				select {
				case udpch <- datagram:
				case <-ctx.Done():
					return
				}
				continue
			}
			tcp, good := players[1].(*layers.TCP)
			if good {
				segment := &tcpSegment{
					packet: packet,
					ipv4:   ipv4,
					tcp:    tcp,
				}
				select {
				case tcpch <- segment:
				case <-ctx.Done():
					return
				}
				continue
			}
			log.Warnf("forwardpath: drop packet: neither UDP nor TCP: %+v", rawPacket)
		case <-ctx.Done():
			return
		}
	}
}
