package main

import (
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

// serve serves requests from a given miniooni client.
func serve(conn net.Conn) {
	defer conn.Close() // we own the conn

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

	// create state for the DNAT
	dnat := &dnatState{
		mu:  sync.Mutex{},
		tcp: map[string]*dnatRecord{},
		udp: map[string]*dnatRecord{},
	}

	// start DNS server running on the user-mode net stack
	dnsConn, err := userNet.ListenUDP(&net.UDPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 53,
		Zone: "",
	})
	if err != nil {
		log.Warnf("userNet.ListenUDP: %s", err.Error())
		return
	}
	defer dnsConn.Close()
	go dnsProxyLoop(dnsConn)

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
	forwardpathRouter(dnat, conn, devTUN)
}
