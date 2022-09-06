package main

import (
	"context"
	"net"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
)

// tcpDevWriter reads raw packets from [inch] and writes them to [dev]. This goroutine
// returns in case of I/O error as well as when [ctx] is done.
func tcpDevWriter(ctx context.Context, inch <-chan []byte, dev tun.Device) {
	for {
		select {
		case rawPacket := <-inch:
			const zeroOffset = 0
			if _, err := dev.Write(rawPacket, zeroOffset); err != nil {
				log.Warnf("tcpDevWriter: dev.Write: %s", err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// tcpDevReader reads raw packets from [dev] and writes them to [outch]. This
// goroutine returns in case of I/O error and when the [ctx] is done.
func tcpDevReader(ctx context.Context, dev tun.Device, outch chan<- []byte) {
	buffer := make([]byte, 4096)
	for {
		const zeroOffset = 0
		count, err := dev.Read(buffer, zeroOffset)
		if err != nil {
			log.Warnf("tcpDevWriter: dev.Read: %s", err.Error())
			return
		}
		rawPacket := buffer[:count]
		select {
		case outch <- rawPacket:
		case <-ctx.Done():
			return
		}
	}
}

// tcpSegmentForwarder reads TCP segments from [inch], applies DNAT policies using [state], and
// posts packets to [outch]. This goroutine returns when the given [ctx] is done.
//
// The DNAT algorithm is as follows:
//
// 1. we use the source port as the DNAT key;
//
// 2. we map the DNAT key to the original destination IP;
//
// 3. we use 10.17.17.1 as the new destination IP.
//
// This function acts on the forward path (probe->internet).
func tcpSegmentForwarder(
	ctx context.Context,
	state *tcpState,
	inch <-chan *tcpSegment,
	outch chan<- []byte,
) {
	for {
		select {
		case segment := <-inch:
			ipv4, tcp := segment.ipv4, segment.tcp
			state.mu.Lock()
			state.m[uint16(tcp.SrcPort)] = ipv4.DstIP // 1 and 2
			state.mu.Unlock()
			ipv4.DstIP = net.IPv4(10, 17, 17, 1) // 3
			pktbuf := gopacket.NewSerializeBuffer()
			opts := gopacket.SerializeOptions{
				FixLengths:       true,
				ComputeChecksums: true,
			}
			tcp.SetNetworkLayerForChecksum(ipv4)
			err := gopacket.SerializeLayers(pktbuf, opts, ipv4, tcp, gopacket.Payload(tcp.Payload))
			runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")
			payload := pktbuf.Bytes()
			select {
			case outch <- payload:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// tcpSegmentBackwarder reads TCP segments from [inch], applies DNAT policies using [state], and
// posts packets to [outch]. This goroutine returns when the given [ctx] is done.
//
// The reverse DNAT algorithm is as follows:
//
// 1. we use the destination port as the DNAT key;
//
// 2. we get the original destination IP using the DNAT key;
//
// 3. we replace the packet's destination IP.
//
// This function acts on the return path (internet->probe).
func tcpSegmentBackwarder(
	ctx context.Context,
	state *tcpState,
	inch <-chan *tcpSegment,
	outch chan<- []byte,
) {
	for {
		select {
		case segment := <-inch:
			ipv4, tcp := segment.ipv4, segment.tcp
			var srcIP net.IP
			state.mu.Lock()
			srcIP = state.m[uint16(tcp.DstPort)] // 1 and 2
			state.mu.Unlock()
			if srcIP == nil {
				log.Warnf("tcpSegmentBackwarder: missing DNAT entry for %d", tcp.DstPort)
				continue
			}
			ipv4.SrcIP = srcIP // 3
			buf := gopacket.NewSerializeBuffer()
			opts := gopacket.SerializeOptions{
				FixLengths:       true,
				ComputeChecksums: true,
			}
			tcp.SetNetworkLayerForChecksum(ipv4)
			err := gopacket.SerializeLayers(buf, opts, ipv4, tcp, gopacket.Payload(tcp.Payload))
			runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")
			payload := buf.Bytes()
			select {
			case outch <- payload:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
