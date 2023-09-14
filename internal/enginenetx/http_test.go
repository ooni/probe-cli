package enginenetx

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPTransport(t *testing.T) {

	// TODO(bassosimone): we should replace this integration test with netemx
	// as soon as we can sever the hard link between netxlite and this pkg
	t.Run("is working as intended", func(t *testing.T) {
		txp := NewHTTPTransport(
			bytecounter.New(), model.DiscardLogger, nil, netxlite.NewStdlibResolver(model.DiscardLogger))
		client := txp.NewHTTPClient()
		resp, err := client.Get("https://www.google.com/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("NewHTTPClient returns a client with a cookie jar", func(t *testing.T) {
		txp := NewHTTPTransport(
			bytecounter.New(), model.DiscardLogger, nil, netxlite.NewStdlibResolver(model.DiscardLogger))
		client := txp.NewHTTPClient()
		if client.Jar == nil {
			t.Fatal("expected non-nil cookie jar")
		}
	})
}
