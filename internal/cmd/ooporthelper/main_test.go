package main

import (
	"context"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestMainWorkingAsIntended(t *testing.T) {
	t.Skip("// TODO(https://github.com/ooni/probe/issues/2338)")
	srvTest = true // toggle to imply that we are running in test mode
	go main()
	dialer := netxlite.NewDialerWithoutResolver(model.DiscardLogger)
	for _, port := range TestPorts {
		<-srvTestChan
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
	}
	srvCancel()  // shutdown server
	srvWg.Wait() // wait for listeners on all ports to close
}
