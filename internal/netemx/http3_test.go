package netemx

import (
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTP3ServerFactory(t *testing.T) {
	t.Run("when using the TLSConfig provided by netem", func(t *testing.T) {
		/*
			 __      ________________________
			/  \    /  \__    ___/\_   _____/
			\   \/\/   / |    |    |    __)
			 \        /  |    |    |     \
			  \__/\  /   |____|    \___  /
			       \/                  \/

			I originally wrote this test to use AddressWwwExampleCom and the test
			failed with generic_timeout_error. Now, instead, if I change it to use
			10.55.56.57, the test is working as intended. I am wondering whether
			I am not fully understanding how quic-go/quic-go works.

			My (limited?) understanding: just a single test can use AddressWwwExampleCom
			and, if I use it in other tests, there are issues leading to timeouts.

			See https://github.com/ooni/probe/issues/2527.
		*/

		env := MustNewQAEnv(
			QAEnvOptionNetStack("10.55.56.57", &HTTP3ServerFactory{
				Factory: HTTPHandlerFactoryFunc(func(_ *netem.UNetStack) http.Handler {
					return ExampleWebPageHandler()
				}),
				Ports:     []int{443},
				TLSConfig: nil, // explicitly nil, let's use netem's config
			}),
		)
		defer env.Close()

		env.AddRecordToAllResolvers("www.example.com", "", "10.55.56.57")

		env.Do(func() {
			client := netxlite.NewHTTP3ClientWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
			req := runtimex.Try1(http.NewRequest("GET", "https://www.example.com/", nil))
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected StatusCode", resp.StatusCode)
			}
			data, err := netxlite.ReadAllContext(req.Context(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(ExampleWebPage, string(data)); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("when using an incompatible TLS config", func(t *testing.T) {
		/*
			 __      ________________________
			/  \    /  \__    ___/\_   _____/
			\   \/\/   / |    |    |    __)
			 \        /  |    |    |     \
			  \__/\  /   |____|    \___  /
			       \/                  \/

			I originally wrote this test to use AddressWwwExampleCom and the test
			failed with generic_timeout_error. Now, instead, if I change it to use
			10.55.56.100, the test is working as intended. I am wondering whether
			I am not fully understanding how quic-go/quic-go works.

			My (limited?) understanding: just a single test can use AddressWwwExampleCom
			and, if I use it in other tests, there are issues leading to timeouts.

			See https://github.com/ooni/probe/issues/2527.
		*/

		// we're creating a distinct MITM TLS config and we're using it, so we expect
		// that we're not able to verify certificates in client code
		mitmConfig := runtimex.Try1(netem.NewTLSMITMConfig())

		env := MustNewQAEnv(
			QAEnvOptionNetStack("10.55.56.100", &HTTP3ServerFactory{
				Factory: HTTPHandlerFactoryFunc(func(_ *netem.UNetStack) http.Handler {
					return ExampleWebPageHandler()
				}),
				Ports:     []int{443},
				TLSConfig: mitmConfig.TLSConfig(), // custom!
			}),
		)
		defer env.Close()

		env.AddRecordToAllResolvers("www.example.com", "", "10.55.56.100")

		env.Do(func() {
			client := netxlite.NewHTTP3ClientWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
			req := runtimex.Try1(http.NewRequest("GET", "https://www.example.com/", nil))
			resp, err := client.Do(req)
			if err == nil || err.Error() != netxlite.FailureSSLInvalidCertificate {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp")
			}
		})
	})
}
