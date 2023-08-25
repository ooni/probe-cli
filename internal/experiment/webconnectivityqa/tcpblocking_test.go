package webconnectivityqa

import (
	"context"
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
		conn, err := dialer.DialContext(context.Background(), "tcp", "130.192.91.7:443")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected error", err)
		}
		if conn != nil {
			t.Fatal("expected to see nil conn")
		}
	})
}
