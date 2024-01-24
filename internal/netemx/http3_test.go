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
		env := MustNewQAEnv(
			QAEnvOptionNetStack(AddressWwwExampleCom, &HTTP3ServerFactory{
				Factory: HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
					return ExampleWebPageHandler()
				}),
				Ports:          []int{443},
				ServerNameMain: "www.example.com",
			}),
		)
		defer env.Close()

		env.AddRecordToAllResolvers("www.example.com", "", AddressWwwExampleCom)

		env.Do(func() {
			netx := &netxlite.Netx{}
			client := netxlite.NewHTTP3ClientWithResolver(netx, log.Log, netx.NewStdlibResolver(log.Log))
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
}
