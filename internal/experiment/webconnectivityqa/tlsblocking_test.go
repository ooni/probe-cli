package webconnectivityqa

import (
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestBlockingTLSConnectionReset(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := tlsBlockingConnectionReset()
	tc.Configure(env)

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)
		req, err := http.NewRequest("GET", "https://www.example.com/", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected error", err)
		}
		if resp != nil {
			t.Fatal("expected nil request")
		}
	})
}
