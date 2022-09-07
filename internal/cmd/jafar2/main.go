package main

import (
	"context"
	"net"
	"net/netip"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// main is the main function
func main() {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{})
	runtimex.PanicOnError(err, "net.ListenTCP")
	log.Infof("listening at %s", listener.Addr().String())
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warnf("Accept: %s", err.Error())
			continue
		}
		serve(conn)
	}
}

// serve serves requests from a given miniooni client [conn].
func serve(conn net.Conn) {
	// make sure we close the conn we own
	defer conn.Close()

	// create context for this request
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// read raw packets in the forward path (miniooni->internet)
	connIn := make(chan []byte)
	go miniooniConnReader(ctx, conn, connIn)

	// write raw packets in the return path (internet->miniooni)
	connOut := make(chan []byte)
	go miniooniConnWriter(ctx, connOut, conn)

	// process incoming raw packets according to protocol in the forward path
	ipv4InUDP := make(chan *udpDatagram)
	ipv4InTCP := make(chan *tcpSegment)
	go ipv4Forwarder(ctx, connIn, ipv4InTCP, ipv4InUDP)

	// create forwarding state for TCP
	tcpState := &tcpState{
		m:  map[uint16]net.IP{},
		mu: sync.Mutex{},
	}

	// apply DNAT rules in the forward path and DNAT to userspace TCP
	tcpDevIn := make(chan []byte)
	go tcpSegmentForwarder(ctx, tcpState, ipv4InTCP, tcpDevIn)

	// special processing rule for DNS-over-UDP
	dnsOverUDPIn := make(chan *udpDatagram)
	go udpDNSHandler(ctx, dnsOverUDPIn, connOut)

	// create usermode network stack for serving requests
	const conservativeMTU = 1250
	devTUN, userNet, err := netstack.CreateNetTUN(
		[]netip.Addr{
			netip.Addr(netip.MustParseAddr("10.17.17.1")),
		},
		[]netip.Addr{
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("8.8.4.4"),
		},
		conservativeMTU,
	)
	if err != nil {
		log.Warnf("netstack.CreateNetTun: %s", err.Error())
		return
	}
	defer devTUN.Close()

	// process incoming raw packets according to protocol in the backward path
	ipv4OutUDP := make(chan *udpDatagram)
	ipv4OutTCP := make(chan *tcpSegment)
	go ipv4Forwarder(ctx)

	// start HTTP listener running on the user-mode net stack
	httpListener, err := userNet.ListenTCP(&net.TCPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 80,
		Zone: "",
	})
	if err != nil {
		log.Warnf("userNet.ListenTCP: %s", err.Error())
		return
	}
	defer httpListener.Close()
	go tcpProxyLoop(dnat, httpListener, "80")

	// start HTTPS listener running on the user-mode net stack
	httpsListener, err := userNet.ListenTCP(&net.TCPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 443,
		Zone: "",
	})
	if err != nil {
		log.Warnf("userNet.ListenTCP: %s", err.Error())
		return
	}
	defer httpsListener.Close()
	go tcpProxyLoop(dnat, httpsListener, "443")

	// start router handling the return path
	go returnpathRouter(dnat, devTUN, conn)

	// run the forward path router in sync fashion
	forwardPathRouter(dnat, conn, devTUN)
}
