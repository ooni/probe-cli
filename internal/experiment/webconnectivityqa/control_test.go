package webconnectivityqa

import (
	"context"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestControlFailureWithSuccessfulHTTPWebsite(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := controlFailureWithSuccessfulHTTPWebsite()
	tc.Configure(env)

	env.Do(func() {
		tcpDialer := netxlite.NewDialerWithStdlibResolver(log.Log)
		tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
		tlsDialer := netxlite.NewTLSDialer(tcpDialer, tlsHandshaker)
		for _, sni := range []string{"0.th.ooni.org", "1.th.ooni.org", "2.th.ooni.org", "3.th.ooni.org", "d33d1gs9kpq1c5.cloudfront.net"} {
			conn, err := tlsDialer.DialTLSContext(context.Background(), "tcp", net.JoinHostPort(sni, "443"))
			if err == nil || err.Error() != netxlite.FailureConnectionReset {
				t.Fatal("unexpected error", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		}
	})
}

func TestControlFailureWithSuccessfulHTTPSWebsite(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := controlFailureWithSuccessfulHTTPSWebsite()
	tc.Configure(env)

	env.Do(func() {
		tcpDialer := netxlite.NewDialerWithStdlibResolver(log.Log)
		tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
		tlsDialer := netxlite.NewTLSDialer(tcpDialer, tlsHandshaker)
		for _, sni := range []string{"0.th.ooni.org", "1.th.ooni.org", "2.th.ooni.org", "3.th.ooni.org", "d33d1gs9kpq1c5.cloudfront.net"} {
			conn, err := tlsDialer.DialTLSContext(context.Background(), "tcp", net.JoinHostPort(sni, "443"))
			if err == nil || err.Error() != netxlite.FailureConnectionReset {
				t.Fatal("unexpected error", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		}
	})
}
