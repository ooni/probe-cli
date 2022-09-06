package main

//
// Censoring DNS proxy
//

import (
	"net"

	"github.com/apex/log"
)

// dnsProxyLoop is the main loop of the DNS proxy.
func dnsProxyLoop(pconn net.PacketConn) {
	buffer := make([]byte, 4096)
	for {
		count, source, err := pconn.ReadFrom(buffer)
		if err != nil {
			log.Warnf("dnsProxyLoop: pconn.ReadFrom failed: %s", err.Error())
			return
		}
		queryPayload := buffer[:count]
		go dnsProxyServe(queryPayload, pconn, source)
	}
}

func dnsProxyServe(queryPayload []byte, clientConn net.PacketConn, source net.Addr) {
	serverConn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Warnf("dnsProxyServe: net.Dial: %s", err.Error())
		return
	}
	defer serverConn.Close()
	if _, err := serverConn.Write(queryPayload); err != nil {
		log.Warnf("dnsProxyServe: Write: %s", err.Error())
		return
	}
	buffer := make([]byte, 4096)
	count, err := serverConn.Read(buffer)
	if err != nil {
		log.Warnf("dnsProxyServe: Read: %s", err.Error())
		return
	}
	responsePayload := buffer[:count]
	if _, err := clientConn.WriteTo(responsePayload, source); err != nil {
		log.Warnf("dnsProxyServe: WriteTo %s", err.Error())
		return
	}
}
