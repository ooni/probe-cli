package netxlite

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func TestNetx(t *testing.T) {
	// create a star network topology
	topology := runtimex.Try1(netem.NewStarTopology(log.Log))
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

	// listen for HTTPS requests using the above handler
	webServerTCPAddress := &net.TCPAddr{
		IP:   net.ParseIP(exampleComAddress),
		Port: 443,
		Zone: "",
	}
	webServerTCPListener := runtimex.Try1(webServerStack.ListenTCP("tcp", webServerTCPAddress))
	webServerTCPServer := &http.Server{
		Handler:   webServerHandler,
		TLSConfig: webServerStack.ServerTLSConfig(),
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
		TLSConfig:  webServerStack.ServerTLSConfig(),
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
