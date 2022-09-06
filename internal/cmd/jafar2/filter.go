package main

import "github.com/google/gopacket/layers"

// filterShouldDropForwardUDPv4 returns whether we should drop this UDPv4 packet.
func filterShouldDropForwardUDPv4(ip *layers.IPv4, udp *layers.UDP) bool {
	return false
}

// filterShouldDropForwardTCPv4 returns whether we should drop this TCPv4 packet.
func filterShouldDropForwardTCPv4(ip *layers.IPv4, tcp *layers.TCP) bool {
	return false
}

// filterShouldDropReturnUDPv4 returns whether we should drop this UDPv4 packet.
func filterShouldDropReturnUDPv4(ip *layers.IPv4, udp *layers.UDP) bool {
	return false
}

// filterShouldDropReturnTCPv4 returns whether we should drop this TCPv4 packet.
func filterShouldDropReturnTCPv4(ip *layers.IPv4, tcp *layers.TCP) bool {
	return false
}
