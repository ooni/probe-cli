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

func TestHTTPSecureServerFactory(t *testing.T) {
	t.Run("when using the TLSConfig provided by netem", func(t *testing.T) {
		env := MustNewQAEnv(
			QAEnvOptionNetStack(AddressWwwExampleCom, &HTTPSecureServerFactory{
				Factory: HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
					return ExampleWebPageHandler()
				}),
				Ports:     []int{443},
				TLSConfig: nil, // explicitly nil, let's use netem's config
			}),
		)
		defer env.Close()

		env.AddRecordToAllResolvers("www.example.com", "", AddressWwwExampleCom)

		env.Do(func() {
			// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
			client := netxlite.NewHTTPClientStdlib(log.Log)
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
		// we're creating a distinct MITM TLS config and we're using it, so we expect
		// that we're not able to verify certificates in client code
		mitmConfig := runtimex.Try1(netem.NewTLSMITMConfig())

		env := MustNewQAEnv(
			QAEnvOptionNetStack(AddressWwwExampleCom, &HTTPSecureServerFactory{
				Factory: HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
					return ExampleWebPageHandler()
				}),
				Ports:     []int{443},
				TLSConfig: mitmConfig.TLSConfig(), // custom!
			}),
		)
		defer env.Close()

		env.AddRecordToAllResolvers("www.example.com", "", AddressWwwExampleCom)

		env.Do(func() {
			// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
			client := netxlite.NewHTTPClientStdlib(log.Log)
			req := runtimex.Try1(http.NewRequest("GET", "https://www.example.com/", nil))
			resp, err := client.Do(req)
			if err == nil || err.Error() != netxlite.FailureSSLUnknownAuthority {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp")
			}
		})
	})
}