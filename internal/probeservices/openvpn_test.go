package probeservices

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func newclientWithStagingEnv() *Client {
	client, err := NewClient(
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		model.OOAPIService{
			Address: "https://api.dev.ooni.io/",
			Type:    "https",
		},
	)
	if err != nil {
		panic(err) // so fail the test
	}
	return client
}

func TestFetchOpenVPNConfig(t *testing.T) {
	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// TODO(ain): switch to newclient() when backend in all environments
		// deploys the vpn-config endpoint.
		clnt := newclientWithStagingEnv()

		// run the tor flow
		config, err := clnt.FetchOpenVPNConfig(context.Background(), "riseup", "ZZ")

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect non-zero length targets
		if len(config.Inputs) <= 0 {
			fmt.Println(config)
			t.Fatal("expected non-zero-length inputs")
		}
	})

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

		// return something that matches thes expected data
		state.SetOpenVPNConfig([]byte(`{
"provider": "demovpn",
"protocol": "openvpn",
"config": {
    "ca": "deadbeef",
    "cert": "deadbeef",
    "key": "deadbeef"
  },
  "date_updated": "2024-05-06T15:22:13.152242Z",
  "endpoints": [
    "openvpn://demovpn.corp/?address=1.1.1.1:53&transport=udp"
  ]
}
`))

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state.NewMux())
		defer srv.Close()

		// TODO(ain)
		client := newclient()

		// override the HTTP client so we speak with our local server rather than the true backend
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// then we can try to fetch the config
		_, err := client.FetchOpenVPNConfig(context.Background(), "demo", "ZZ")

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}
	})
}
