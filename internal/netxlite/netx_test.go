package netxlite

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// This test ensures that a Netx wrapping a netem.UNet is WAI
func TestNetxWithNetem(t *testing.T) {
	// create a star network topology
	topology := netem.MustNewStarTopology(log.Log)
	defer topology.Close()

	// constants for the IP address we're using
	const (
		clientAddress     = "130.192.91.211"
		exampleComAddress = "93.184.216.34"
		quad8Address      = "8.8.8.8"
	)

	// create and configure the name server
	nameServerStack := runtimex.Try1(topology.AddHost(quad8Address, quad8Address, &netem.LinkConfig{}))
	nameServerConfig := netem.NewDNSConfig()
	nameServerConfig.AddRecord("www.example.com", "web01.example.com", exampleComAddress)
	nameServer := runtimex.Try1(netem.NewDNSServer(log.Log, nameServerStack, quad8Address, nameServerConfig))
	defer nameServer.Close()

	// create the web server handler
	bonsoirElliot := []byte("Bonsoir, Elliot!\r\n")
	webServerStack := runtimex.Try1(topology.AddHost(exampleComAddress, quad8Address, &netem.LinkConfig{}))
	webServerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bonsoirElliot)
	})

	// create common certificate for HTTPS and HTTP3
	webServerTLSConfig := webServerStack.MustNewServerTLSConfig("www.example.com", "web01.example.com")

	// listen for HTTPS requests using the above handler
	webServerTCPAddress := &net.TCPAddr{
		IP:   net.ParseIP(exampleComAddress),
		Port: 443,
		Zone: "",
	}
	webServerTCPListener := runtimex.Try1(webServerStack.ListenTCP("tcp", webServerTCPAddress))
	webServerTCPServer := &http.Server{
		Handler:   webServerHandler,
		TLSConfig: webServerTLSConfig,
	}
	go webServerTCPServer.ServeTLS(webServerTCPListener, "", "")
	defer webServerTCPServer.Close()

	// listen for HTTP/3 requests using the above handler
	webServerUDPAddress := &net.UDPAddr{
		IP:   net.ParseIP(exampleComAddress),
		Port: 443,
		Zone: "",
	}
	webServerUDPListener := runtimex.Try1(webServerStack.ListenUDP("udp", webServerUDPAddress))
	webServerUDPServer := &http3.Server{
		TLSConfig:  webServerTLSConfig,
		QuicConfig: &quic.Config{},
		Handler:    webServerHandler,
	}
	go webServerUDPServer.Serve(webServerUDPListener)
	defer webServerUDPServer.Close()

	// create the client userspace TCP/IP stack and the corresponding netx
	clientStack := runtimex.Try1(topology.AddHost(clientAddress, quad8Address, &netem.LinkConfig{}))
	underlyingNetwork := &NetemUnderlyingNetworkAdapter{clientStack}
	netx := &Netx{underlyingNetwork}

	t.Run("HTTPS fetch", func(t *testing.T) {
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPTransportStdlib is QUIRKY but we probably
		// don't care about using a QUIRKY function here?
		txp := netx.NewHTTPTransportStdlib(log.Log)
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.example.com/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code")
		}
		body, err := ReadAllContext(context.Background(), resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(bonsoirElliot, body); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("HTTP/3 fetch", func(t *testing.T) {
		txp := netx.NewHTTP3TransportStdlib(log.Log)
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.example.com/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code")
		}
		body, err := ReadAllContext(context.Background(), resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(bonsoirElliot, body); diff != "" {
			t.Fatal(diff)
		}
	})
}

// We generally do not listen here as part of other tests, since the listening
// functionality is mainly only use for testingx. So, here's a specific test for that.
func TestNetxListenTCP(t *testing.T) {
	netx := &Netx{Underlying: nil}

	listener := runtimex.Try1(netx.ListenTCP("tcp", &net.TCPAddr{}))
	serverEndpoint := listener.Addr().String()

	// listen in a background goroutine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		conn := runtimex.Try1(listener.Accept())
		conn.Close()
		wg.Done()
	}()

	// dial in a background goroutine
	wg.Add(1)
	go func() {
		ctx := context.Background()
		dialer := netx.NewDialerWithoutResolver(log.Log)
		conn := runtimex.Try1(dialer.DialContext(ctx, "tcp", serverEndpoint))
		conn.Close()
		wg.Done()
	}()

	// wait for the goroutines to finish
	wg.Wait()
}
