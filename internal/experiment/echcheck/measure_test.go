package echcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "echcheck" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected version")
	}
}

// qaenv creates a [netemx.QAEnv] with a single crypto.cloudflare.com test server and a DoH server.
func qaenv() *netemx.QAEnv {
	cfg := []*netemx.ScenarioDomainAddresses{
		{
			Domains:          []string{"crypto.cloudflare.com"},
			Addresses:        []string{"130.192.91.7"},
			Role:             netemx.ScenarioRoleWebServer,
			WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
		},
		{
			Domains:   []string{"mozilla.cloudflare-dns.com"},
			Addresses: []string{"130.192.91.13"},
			Role:      netemx.ScenarioRolePublicDNS,
		},
	}
	return netemx.MustNewScenario(cfg)
}

func TestMeasurerMeasureWithCancelledContext(t *testing.T) {
	// create QAEnv
	env := qaenv()
	defer env.Close()

	env.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel the context

		// create measurer
		measurer := NewExperimentMeasurer(Config{})
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}

		// run measurement
		err := measurer.Run(ctx, args)
		if err == nil {
			t.Fatal("expected an error here")
		}
		if err.Error() != "interrupted" {
			t.Fatal("unexpected error type")
		}
	})

}

func TestMeasurerMeasureWithInvalidInput(t *testing.T) {
	// create QAEnv
	env := qaenv()
	defer env.Close()

	// create measurer
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks: model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: &model.Measurement{
			// leading space to test url.Parse failure
			Input: " https://crypto.cloudflare.com/cdn-cgi/trace",
		},
		Session: &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
	}
	// run measurement
	err := measurer.Run(context.Background(), args)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "input is not an URL" {
		t.Fatal("unexpected error type")
	}
}

func TestMeasurementSuccessRealWorld(t *testing.T) {
	if testing.Short() {
		// this test uses the real internet so we want to skip this in short mode
		t.Skip("skip test in short mode")
	}

	// create measurer
	measurer := NewExperimentMeasurer(Config{})
	msrmnt := &model.Measurement{}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: msrmnt,
		Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
	}

	// run measurement
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	// check results
	tk := msrmnt.TestKeys.(TestKeys)
	if len(tk.Queries) != 3 {
		// TODO Check that expected DNS Queries are included
		t.Fatal("unexpected number of DNS Queries recorded", len(tk.Queries))
	}
	if len(tk.NetworkEvents) != 4 {
		t.Fatal("unexpected number of Network events recorded", len(tk.NetworkEvents))
	}
	// NoECH, GREASE, GREASE retry, RealECH
	if len(tk.TLSHandshakes) != 4 {
		t.Fatal("unexpected number of TLS handshakes", len(tk.TLSHandshakes))
	}
	if len(tk.TCPConnects) != 4 {
		t.Fatal("unexpected number of TCP connections", len(tk.TCPConnects))
	}
	for _, hs := range tk.TLSHandshakes {
		if hs.Failure != nil {
			if hs.ECHConfig == "GREASE" {
				// We expect that this either succeeds (i.e. with a non-ECH server)
				// OR that it fails with an EchRejeECHRejectionError
				if !strings.Contains(*hs.Failure, "tls: server rejected ECH") {
					t.Fatal("unexpected exp (grease) failure:", *hs.Failure)
				}
			} else if len(hs.ECHConfig) > 0 {
				t.Fatal("unexpected exp (ech) failure:", *hs.Failure)
			} else {
				t.Fatal("unexpected ctrl failure:", *hs.Failure)
			}
		}
	}
}
