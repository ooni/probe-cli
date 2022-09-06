package main

import (
	"net"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// tcpSegment is an incoming TCP segment.
type tcpSegment struct {
	// packet is the original packet
	packet gopacket.Packet

	// ipv4 is the IPv4 layer
	ipv4 *layers.IPv4

	// tcp is the TCP layer
	tcp *layers.TCP
}

// udpDatagram is an incoming UDP datagram
type udpDatagram struct {
	// packet is the original packet
	packet gopacket.Packet

	// ipv4 is the IPv4 layer
	ipv4 *layers.IPv4

	// udp is the UDP layer
	udp *layers.UDP
}

// tcpState contains state for TCP DNAT.
type tcpState struct {
	// m maps state keys to the source address
	m map[uint16]net.IP

	// mu provides mutual exclusion
	mu sync.Mutex
}
