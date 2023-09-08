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

func TestHTTPCleartextServerFactory(t *testing.T) {
	env := MustNewQAEnv(
		QAEnvOptionNetStack(AddressWwwExampleCom, &HTTPCleartextServerFactory{
			Factory: HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
				return ExampleWebPageHandler()
			}),
			Ports: []int{80},
		}),
	)
	defer env.Close()

	env.AddRecordToAllResolvers("www.example.com", "", AddressWwwExampleCom)

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)
		req := runtimex.Try1(http.NewRequest("GET", "http://www.example.com/", nil))
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
}
