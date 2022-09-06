package main

import (
	"context"
	"net"
	"net/netip"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

func main() {
	// create socket for reading user packets
	devUDP, err := net.ListenUDP("udp", &net.UDPAddr{})
	runtimex.PanicOnError(err, "net.ListenUDP")
	localAddr := devUDP.LocalAddr()
	log.Infof("listening at %s/%s", localAddr.String(), localAddr.Network())

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
	runtimex.PanicOnError(err, "netstack.CreateNetTun failed")

	// start DNS server running on the user-mode net stack
	dnsConn, err := userNet.ListenUDP(&net.UDPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 53,
		Zone: "",
	})
	runtimex.PanicOnError(err, "userNet.ListenUDP")
	go dnsProxyLoop(dnsConn)

	// start the forward path loop
	go fwdPathLoop(devUDP, devTUN)

	// start the return path loop
	go returnPathLoop(devTUN, devUDP)

	// wait forever
	<-context.Background().Done()
}
