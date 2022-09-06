package main

//
// Censoring DNS proxy
//

import (
	"net"

	"github.com/apex/log"
	"github.com/miekg/dns"
)

// dnsProxyLoop is the main loop of the DNS proxy.
func dnsProxyLoop(pconn net.PacketConn) {
	buffer := make([]byte, 4096)
	for {
		count, source, err := pconn.ReadFrom(buffer)
		if err != nil {
			log.Warnf("jafar-proxy: dnsProxyLoop: pconn.ReadFrom failed: %s", err.Error())
			continue
		}
		queryPayload := buffer[:count]
		query := &dns.Msg{}
		if err := query.Unpack(queryPayload); err != nil {
			log.Warnf("jafar-proxy: dnsProxyLoop: query.Unpack failed: %s", err.Error())
			continue
		}
		// TODO(bassosimone): here we should do a bit more than reply NXDOMAIN
		response := &dns.Msg{}
		response.SetReply(query)
		response.Rcode = dns.RcodeNameError
		responsePayload, err := response.Pack()
		if err != nil {
			log.Warnf("jafar-proxy: dnsProxyLoop: response.Pack failed: %s", err.Error())
			continue
		}
		if _, err = pconn.WriteTo(responsePayload, source); err != nil {
			log.Warnf("jafar-proxy: dnsProxyLoop: pconn.WriteTo failed: %s", err.Error())
			continue
		}
	}
}
