package webconnectivityqa

import (
	"context"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTCPBlockingConnectTimeout(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := tcpBlockingConnectTimeout()
	tc.Configure(env)

	env.Do(func() {
		dialer := netxlite.NewDialerWithoutResolver(log.Log)
		endpoint := net.JoinHostPort(netemx.AddressWwwExampleCom, "443")
		conn, err := dialer.DialContext(context.Background(), "tcp", endpoint)
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected to see nil conn")
		}
	})
}

func TestTCPBlockingConnectionRefusedWithInconsistentDNS(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := tcpBlockingConnectionRefusedWithInconsistentDNS()
	tc.Configure(env)

	env.Do(func() {
		dialer := netxlite.NewDialerWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
		conn, err := dialer.DialContext(context.Background(), "tcp", "www.example.org:443")
		if err == nil || err.Error() != netxlite.FailureConnectionRefused {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected to see nil conn")
		}
	})
}
