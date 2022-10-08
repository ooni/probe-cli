package main

import (
	"context"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

var (
	portsMap = make(map[string]bool)
)

func TestMainWorkingAsIntended(t *testing.T) {
	srvTest = true // toggle to imply that we are running in test mode
	for _, port := range TestPorts {
		portsMap[port] = false
	}
	go main()
	dialer := netxlite.NewDialerWithoutResolver(model.DiscardLogger)
	for i := 0; i < len(TestPorts); i++ {
		port := <-srvTestChan
		addr := net.JoinHostPort("127.0.0.1", port)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		if conn == nil {
			t.Fatal("expected non-nil conn")
		}
		conn.Close()
		portsMap[port] = true
	}
	srvCancel()  // shutdown server
	srvWg.Wait() // wait for listeners on all ports to close
	// check if all ports were covered
	for _, port := range TestPorts {
		if !portsMap[port] {
			t.Fatal("missed port in test", port)
		}
	}
}
