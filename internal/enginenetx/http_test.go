package enginenetx_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPTransport(t *testing.T) {

	t.Run("the HTTPTransport is working as intended", func(t *testing.T) {
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		env.Do(func() {
			txp := enginenetx.NewHTTPTransport(
				bytecounter.New(), model.DiscardLogger, nil, netxlite.NewStdlibResolver(model.DiscardLogger))
			client := txp.NewHTTPClient()
			resp, err := client.Get("https://www.example.com/")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected status code")
			}
		})
	})

	t.Run("we can use a socks5 proxy", func(t *testing.T) {
		panic("not implemented")
	})

	t.Run("we can use an HTTP proxy", func(t *testing.T) {
		panic("not implemented")
	})

	t.Run("we can use an HTTPS proxy", func(t *testing.T) {
		panic("not implemented")
	})

	t.Run("NewHTTPClient returns a client with a cookie jar", func(t *testing.T) {
		txp := enginenetx.NewHTTPTransport(
			bytecounter.New(), model.DiscardLogger, nil, netxlite.NewStdlibResolver(model.DiscardLogger))
		client := txp.NewHTTPClient()
		if client.Jar == nil {
			t.Fatal("expected non-nil cookie jar")
		}
	})
}
