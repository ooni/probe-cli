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
	type testCaseConfig struct {
		expectedErr string
		testCase    *TestCase
	}

	testcases := []*testCaseConfig{{
		expectedErr: "ssl_unknown_authority",
		testCase:    badSSLWithUnknownAuthorityWithConsistentDNS(),
	}, {
		expectedErr: "ssl_invalid_certificate",
		testCase:    badSSLWithExpiredCertificate(),
	}, {
		expectedErr: "ssl_invalid_hostname",
		testCase:    badSSLWithWrongServerName(),
	}, {
		expectedErr: "ssl_unknown_authority",
		testCase:    badSSLWithUnknownAuthorityWithInconsistentDNS(),
	}}

	for _, tc := range testcases {
		t.Run(tc.testCase.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.testCase.Configure(env)

			env.Do(func() {
				// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
				client := netxlite.NewHTTPClientStdlib(log.Log)
				req := runtimex.Try1(http.NewRequest("GET", tc.testCase.Input, nil))
				resp, err := client.Do(req)
				if err == nil || err.Error() != tc.expectedErr {
					t.Fatal("unexpected err", err)
				}
				if resp != nil {
					t.Fatal("expected nil resp")
				}
			})
		})
	}
}
