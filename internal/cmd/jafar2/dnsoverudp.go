package main

//
// DNS-over-UDPv4 proxy
//

import (
	"context"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// dnsOverUDPv4 is the entry point of the DNS over UDPv4 proxy.
//
// Arguments:
//
// - ctx is the context binding the lifetime of this goroutine;
//
// - wg is the wait group used by the parent;
//
// - clientConn is the TCP conn with the client (usually miniooni);
//
// - tcpState is the TCP state for implementing DNAT;
//
// - packetsForClient is the queue where to append packets for the client;
//
// - tcpDev is the virtual device connected to TCP servers.
//
// This function may spawn off a goroutine if there's network I/O to perform.
func dnsOverUDPv4(
	ctx context.Context,
	rawPacket []byte,
	ipv4 *layers.IPv4,
	udp *layers.UDP,
	packetsForClient chan<- []byte,
) {
	// read the incoming query
	query := &dns.Msg{}
	if err := query.Unpack(udp.Payload); err != nil {
		log.Warnf("dnsOverUDPv4: drop packet: not a DNS packet: %+v", rawPacket)
		return
	}

	// it seems we want to forward this query to a remote
	// server, therefore let's do this async
	go dnsOverUDPv4Worker(ctx, ipv4, udp, packetsForClient, query)
}

// dnsOverUDPv4Worker is the background worker for DNS-over-IPv4 queries. This
// goroutine runs until [ctx] is done or it has completed its job.
func dnsOverUDPv4Worker(
	ctx context.Context,
	ipv4 *layers.IPv4,
	udp *layers.UDP,
	packetsForClient chan<- []byte,
	query *dns.Msg,
) {
	// send the query to an upstream server
	resp, err := dns.Exchange(query, "8.8.8.8:53")
	if err != nil {
		log.Warnf("dnsOverUDPv4Worker: dns.Exchange: %s", err.Error())
		return
	}
	rawResp, err := resp.Pack()
	if err != nil {
		log.Warnf("dnsOverUDPv4Worker: resp.Pack: %s", err.Error())
		return
	}

	// assemble a response for the client
	ipv4.SrcIP, ipv4.DstIP = ipv4.SrcIP, ipv4.DstIP
	udp.SrcPort, udp.DstPort = udp.DstPort, udp.SrcPort
	pktbuf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	udp.SetNetworkLayerForChecksum(ipv4)
	err = gopacket.SerializeLayers(pktbuf, opts, ipv4, udp, gopacket.Payload(rawResp))
	runtimex.PanicOnError(err, "gopacket.SerializeLayers failed")
	payload := pktbuf.Bytes()

	// queue the response for the client to process it
	select {
	case <-ctx.Done():
	case packetsForClient <- payload:
	}
}
