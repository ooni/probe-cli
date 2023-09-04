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
		endpoint := net.JoinHostPort(netemx.InternetScenarioAddressWwwExampleCom, "443")
		conn, err := dialer.DialContext(context.Background(), "tcp", endpoint)
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected to see nil conn")
		}
	})
}
