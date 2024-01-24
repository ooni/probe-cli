package netemx

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestBadSSL(t *testing.T) {
	env := MustNewScenario(InternetScenario)
	defer env.Close()

	env.Do(func() {

		// testcase is a testcase supported by this function
		type testcase struct {
			serverName string
			expectErr  string
		}

		testcases := []testcase{{
			serverName: "untrusted-root.badssl.com",
			expectErr:  netxlite.FailureSSLUnknownAuthority,
		}, {
			serverName: "wrong.host.badssl.com",
			expectErr:  netxlite.FailureSSLInvalidHostname,
		}, {
			serverName: "expired.badssl.com",
			expectErr:  netxlite.FailureSSLInvalidCertificate,
		}, {
			// Make sure that we can use the badssl server as something we can
			// force using the DNS to cause a failure
			serverName: "www.example.com",
			expectErr:  netxlite.FailureSSLUnknownAuthority,
		}}

		for _, tc := range testcases {
			t.Run(fmt.Sprintf("for %s expect %s", tc.serverName, tc.expectErr), func(t *testing.T) {
				tlsConfig := &tls.Config{ServerName: tc.serverName}

				netx := &netxlite.Netx{}
				tlsDialer := netxlite.NewTLSDialerWithConfig(
					netx.NewDialerWithoutResolver(log.Log),
					netxlite.NewTLSHandshakerStdlib(log.Log),
					tlsConfig,
				)

				endpoint := net.JoinHostPort(AddressBadSSLCom, "443")
				conn, err := tlsDialer.DialTLSContext(context.Background(), "tcp", endpoint)
				if err == nil || err.Error() != tc.expectErr {
					t.Fatal("unexpected error", err)
				}
				if conn != nil {
					t.Fatal("expected nil conn")
				}
			})
		}
	})
}
