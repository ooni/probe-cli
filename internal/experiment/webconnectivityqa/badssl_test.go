package webconnectivityqa

import (
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestBadSSLConditions(t *testing.T) {
	testcases := map[string]*TestCase{
		"ssl_unknown_authority":   badSSLWithUnknownAuthority(),
		"ssl_invalid_certificate": badSSLWithExpiredCertificate(),
		"ssl_invalid_hostname":    badSSLWithWrongServerName(),
	}

	for expectedErr, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			tc.Configure(env)

			env.Do(func() {
				client := netxlite.NewHTTPClientStdlib(log.Log)
				req := runtimex.Try1(http.NewRequest("GET", tc.Input, nil))
				resp, err := client.Do(req)
				if err == nil || err.Error() != expectedErr {
					t.Fatal("unexpected err", err)
				}
				if resp != nil {
					t.Fatal("expected nil resp")
				}
			})
		})
	}
}
