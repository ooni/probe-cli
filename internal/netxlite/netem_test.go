package netxlite

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestNetemUnderlyingNetworkAdapter(t *testing.T) {

	// This test case explicitly ensures we can use the adapter to listen for TCP
	t.Run("ListenTCP", func(t *testing.T) {
		// create a star network topology
		topology := runtimex.Try1(netem.NewStarTopology(log.Log))
		defer topology.Close()

		// constants for the IP address we're using
		const (
			clientAddress = "130.192.91.211"
			serverAddress = "93.184.216.34"
		)

		// create the stacks
		serverStack := runtimex.Try1(topology.AddHost(serverAddress, "0.0.0.0", &netem.LinkConfig{}))
		clientStack := runtimex.Try1(topology.AddHost(clientAddress, "0.0.0.0", &netem.LinkConfig{}))

		// wrap the server stack and create listening socket
		serverAdapter := &NetemUnderlyingNetworkAdapter{serverStack}
		serverEndpoint := &net.TCPAddr{IP: net.ParseIP(serverAddress), Port: 54321}
		listener := runtimex.Try1(serverAdapter.ListenTCP("tcp", serverEndpoint))
		defer listener.Close()

		// listen in a background goroutine
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			conn := runtimex.Try1(listener.Accept())
			conn.Close()
			wg.Done()
		}()

		// wrap the client stack
		clientAdapter := &NetemUnderlyingNetworkAdapter{clientStack}

		// connect in a background goroutine
		wg.Add(1)
		go func() {
			ctx := context.Background()
			conn := runtimex.Try1(clientAdapter.DialContext(ctx, "tcp", serverEndpoint.String()))
			conn.Close()
			wg.Done()
		}()

		// wait for all operations to complete
		wg.Wait()
	})

}
