package main

import (
	"context"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// udpPortClassifier reads UDP datagrams from [inch] and passes DNS queries to the
// [dnsch] channel. This function returns when [ctx] is done.
//
// Note that this function just discards all unknown UDP packets.
func udpPortClassifier(
	ctx context.Context,
	inch <-chan *udpDatagram,
	dnsch chan<- *udpDatagram,
) {
	for {
		select {
		case datagram := <-inch:
			udp := datagram.udp
			switch udp.DstPort {
			case 53:
				dnsch <- datagram
			default:
				// nothing
			}
		case <-ctx.Done():
			return
		}
	}
}

// udpDNSHandler reads DNS query candidates from [inch], performs the query, and
// a serialized response back to [outch]. This function returns when [ctx] is done.
func udpDNSHandler(ctx context.Context, inch <-chan *udpDatagram, outch chan<- []byte) {
	for {
		select {
		case datagram := <-inch:
			ipv4, udp := datagram.ipv4, datagram.udp
			rawQuery := udp.Payload
			query := &dns.Msg{}
			if err := query.Unpack(rawQuery); err != nil {
				log.Warnf("udpDNSHandler: query.Unpack: %s", err.Error())
				continue
			}
			resp, err := dns.Exchange(query, "8.8.8.8:53")
			if err != nil {
				log.Warnf("udpDNSHandler: dns.Exchange: %s", err.Error())
				continue
			}
			rawResp, err := resp.Pack()
			if err != nil {
				log.Warnf("udpDNSHandler: resp.Pack: %s", err.Error())
				continue
			}
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
