package main

//
// TCP proxy
//

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
)

// tcpProxyLoop is the loop associated with a TCP proxy.
func tcpProxyLoop(dnat *dnatState, listener net.Listener, localPort string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warnf("tcpProxyLoop: Accept: %s", err.Error())
			return
		}
		go tcpProxyServe(dnat, conn, localPort)
	}
}

// tcpProxyServe serves a given conn
func tcpProxyServe(dnat *dnatState, conn net.Conn, localPort string) {
	defer conn.Close() // we own the conn

	// step 1: obtain the four tuple
	srcIP, srcPort, dstIP, dstPort, err := fourTuple(conn)
	if err != nil {
		log.Warnf("tcpProxyServe: fourTuple: %s", err.Error())
		return

	}

	// 2. use DNAT to get the real destination addr
	rec, err := dnat.getRecord(
		uint8(layers.IPProtocolTCP),
		srcIP,
		srcPort,
		dstIP,
		dstPort,
	)
	if err != nil {
		log.Warnf("tcpProxyServe: dnat.getRecord: %s", err.Error())
		return
	}

	// 3. compute the remote endpoint
	endpoint := net.JoinHostPort(rec.origDstIP.String(), localPort)

	// 4. dial the connection
	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	realConn, err := dialer.DialContext(context.Background(), "tcp", endpoint)
	if err != nil {
		log.Warnf("tcpProxyServer: dialer.DialContext: %s", err.Error())
		return
	}
	defer realConn.Close()

	// 5. pipe the two connections
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go tcpProxyReadWrite(wg, conn, realConn)
	go tcpProxyReadWrite(wg, realConn, conn)

	// 6. wait for termination
	wg.Wait()
}

// tcpProxyReadWrite reads from left and writes to right
func tcpProxyReadWrite(wg *sync.WaitGroup, left, right net.Conn) {
	defer wg.Done()
	io.Copy(left, right)
}
