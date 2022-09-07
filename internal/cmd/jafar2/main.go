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
		clientConn, err := listener.Accept()
		if err != nil {
			log.Warnf("main: Accept: %s", err.Error())
			continue
		}
		go serve(clientConn)
	}
}

// serve serves requests from a given miniooni client [conn].
func serve(clientConn net.Conn) {
	// make sure we close the conn we own
	defer clientConn.Close()

	// create context for this request
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// know when all background services terminated
	wg := &sync.WaitGroup{}

	// create forwarding state for TCP
	tcpState := &tcpState{
		dnat: map[uint16]net.IP{},
		mu:   sync.Mutex{},
	}

	// make queue for packets in the return path
	packetsForClient := make(chan []byte, 128)

	// create usermode network stack for serving requests
	const conservativeMTU = 1250
	tcpDev, userNet, err := netstack.CreateNetTUN(
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
		log.Warnf("serve: netstack.CreateNetTun: %s", err.Error())
		return
	}
	defer tcpDev.Close()

	// process packets in the forward path (miniooni->proxies)
	wg.Add(1)
	go clientConnReader(
		ctx,
		wg,
		clientConn,
		tcpState,
		packetsForClient,
		tcpDev,
	)

	// process packets in the return path (miniooni<-proxies)
	wg.Add(1)
	go tcpDeviceReader(
		ctx,
		wg,
		tcpState,
		tcpDev,
		packetsForClient,
	)

	// actually forward packets to clients
	wg.Add(1)
	go clientConnWriter(
		ctx,
		wg,
		packetsForClient,
		clientConn,
	)

	// start HTTP listener running on the user-mode net stack
	httpListener, err := userNet.ListenTCP(&net.TCPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 80,
		Zone: "",
	})
	if err != nil {
		log.Warnf("serve: userNet.ListenTCP: %s", err.Error())
		return
	}
	defer httpListener.Close()
	wg.Add(1)
	go tcpProxyLoop(
		ctx,
		wg,
		tcpState,
		httpListener,
		"80",
	)

	// start HTTPS listener running on the user-mode net stack
	httpsListener, err := userNet.ListenTCP(&net.TCPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 443,
		Zone: "",
	})
	if err != nil {
		log.Warnf("serve: userNet.ListenTCP: %s", err.Error())
		return
	}
	defer httpsListener.Close()
	wg.Add(1)
	go tcpProxyLoop(
		ctx,
		wg,
		tcpState,
		httpsListener,
		"443",
	)

	// block until goroutines terminate
	wg.Wait()
}
