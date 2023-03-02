package netem3

//
// Code to dump packets
//

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
)

// maybeDumpPacket dumps a packet if the enabled flag is true. We will dump
// packets using the github.com/apex/log default logger.
func maybeDumpPacket(enabled bool, nicName string, rawPacket []byte) {
	if enabled {
		dumpPacket(nicName, rawPacket)
	}
}

// dumpPacket dumps a packet. We will dump
// packets using the github.com/apex/log default logger.
func dumpPacket(nicName string, rawPacket []byte) {
	// decode the packet as IPv4
	packet, err := dissectPacket(rawPacket)
	if err != nil {
		log.Warnf("netem: dumpPacket: %s", err.Error())
		return
	}

	// write information about the NIC
	output := &strings.Builder{}
	fmt.Fprintf(output, "netem: %s: ", nicName)

	// write information about the TCP/IP layer
	switch {
	case packet.tcp != nil:
		fmt.Fprintf(
			output,
			"TCP %s.%d -> %s.%d: flags %s, seq %d, ack %d, ttl %d, length %d",
			packet.sourceIPAddress(),
			packet.tcp.SrcPort,
			packet.destinationIPAddress(),
			packet.tcp.DstPort,
			dumpFormatTCPFlags(packet.tcp),
			packet.tcp.Seq,
			packet.tcp.Ack,
			packet.timeToLive(),
			len(packet.tcp.Payload),
		)

	case packet.udp != nil:
		fmt.Fprintf(
			output,
			"UDP %s.%d -> %s.%d: ttl %d, length %d",
			packet.sourceIPAddress(),
			packet.udp.SrcPort,
			packet.destinationIPAddress(),
			packet.udp.DstPort,
			packet.timeToLive(),
			len(packet.udp.Payload),
		)

	default:
		fmt.Fprintf(output, "<unknown>")
	}

	log.Info(output.String())
}

// dumpFormatTCPFlags formats TCP flags as a string.
func dumpFormatTCPFlags(tcp *layers.TCP) string {
	output := &strings.Builder{}
	fmt.Fprintf(output, "[")

	if tcp.ACK {
		fmt.Fprintf(output, "A")
	} else {
		fmt.Fprintf(output, ".")
	}

	if tcp.PSH {
		fmt.Fprintf(output, "P")
	} else {
		fmt.Fprintf(output, ".")
	}

	if tcp.RST {
		fmt.Fprintf(output, "R")
	} else {
		fmt.Fprintf(output, ".")
	}

	if tcp.SYN {
		fmt.Fprintf(output, "S")
	} else {
		fmt.Fprintf(output, ".")
	}

	if tcp.FIN {
		fmt.Fprintf(output, "F")
	} else {
		fmt.Fprintf(output, ".")
	}

	fmt.Fprintf(output, "]")
	return output.String()
}
