package telegram_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := telegram.NewExperimentMeasurer(telegram.Config{})
	if measurer.ExperimentName() != "telegram" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected version")
	}
}

func TestUpdateWithNoAccessPointsBlocking(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: nil, // this should be enough to declare success
		},
	})
	if tk.TelegramHTTPBlocking == true {
		t.Fatal("there should be no TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithNilFailedOperation(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithNonConnectFailedOperation(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.HTTPRoundTripOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithAllConnectsFailed(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == false {
		t.Fatal("there should be TelegramTCPBlocking")
	}
}

func TestUpdateWithWebFailure(t *testing.T) {
	failure := netxlite.FailureEOFError
	failedOperation := netxlite.TLSHandshakeOperation
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://web.telegram.org/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &failedOperation,
		},
	})
	if tk.TelegramWebStatus != "blocked" {
		t.Fatal("TelegramWebStatus should be blocked")
	}
	if *tk.TelegramWebFailure != failure {
		t.Fatal("invalid TelegramWebFailure")
	}
}

func TestUpdateWithAllGood(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://web.telegram.org/",
		},
		TestKeys: urlgetter.TestKeys{
			HTTPResponseStatus: 200,
			HTTPResponseBody:   "<HTML><title>Telegram Web</title></HTML>",
		},
	})
	if tk.TelegramWebStatus != "ok" {
		t.Fatal("TelegramWebStatus should be ok")
	}
	if tk.TelegramWebFailure != nil {
		t.Fatal("invalid TelegramWebFailure")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &telegram.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	failure := io.EOF.Error()
	tests := []struct {
		tk        telegram.TestKeys
		isAnomaly bool
	}{{
		tk:        telegram.TestKeys{},
		isAnomaly: false,
	}, {
		tk:        telegram.TestKeys{TelegramTCPBlocking: true},
		isAnomaly: true,
	}, {
		tk:        telegram.TestKeys{TelegramHTTPBlocking: true},
		isAnomaly: true,
	}, {
		tk:        telegram.TestKeys{TelegramWebFailure: &failure},
		isAnomaly: true,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &telegram.Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(telegram.SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
		})
	}
}

// Creates an experiment-specific configuration for the [netemx.Environment].
func envConfig() netemx.Config {
	// create the default configuration for DNS server
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord(
		"web.telegram.org",
		"web.telegram.org", // CNAME
		"149.154.167.99",
	)
	return envConfigWithDNS(dnsConfig)
}

// Creates an experiment-specific configuration for the [netemx.Environment]
// with custom DNS.
func envConfigWithDNS(dnsConfig *netem.DNSConfig) netemx.Config {
	// config for the telegram Web server stack
	telegramWeb := netemx.ServerStack{
		ServerAddr: "149.154.167.99",
		Listeners:  []netemx.Listener{{Port: 443}},
	}
	servers := []netemx.ServerStack{telegramWeb}
	// for each datacenter we configure a server stack, running a port 443 and 80 instance each
	for _, dc := range telegram.Datacenters {
		servers = append(servers, netemx.ServerStack{
			ServerAddr: dc,
			Listeners:  []netemx.Listener{{Port: 443}, {Port: 80}},
		})
	}
	return netemx.Config{
		DNSConfig: dnsConfig,
		Servers:   servers,
	}
}

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
}

func TestMeasurerRun(t *testing.T) {

	t.Run("Test Measurer without DPI: expect success", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*telegram.TestKeys)
			if tk.TelegramWebFailure != nil {
				t.Fatalf("Unexpected Telegram Web failure %s", *tk.TelegramWebFailure)
			}
			if tk.TelegramHTTPBlocking {
				t.Fatalf("Unexpected HTTP blocking")
			}
			if tk.TelegramTCPBlocking {
				t.Fatal("Unexpected TCP blocking")
			}
			if tk.Agent != "redirect" {
				t.Fatal("unexpected Agent")
			}
			if tk.FailedOperation != nil {
				t.Fatal("unexpected FailedOperation")
			}
			if tk.Failure != nil {
				t.Fatal("unexpected Failure")
			}
			if len(tk.NetworkEvents) <= 0 {
				t.Fatal("no NetworkEvents?!")
			}
			if len(tk.Queries) <= 0 {
				t.Fatal("no Queries?!")
			}
			if len(tk.Requests) <= 0 {
				t.Fatal("no Requests?!")
			}
			if len(tk.TCPConnect) <= 0 {
				t.Fatal("no TCPConnect?!")
			}
			if len(tk.TLSHandshakes) <= 0 {
				t.Fatal("no TLSHandshakes?!")
			}
			if tk.TelegramWebStatus != "ok" {
				t.Fatal("unexpected TelegramWebStatus")
			}
			sk, err := measurer.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := sk.(telegram.SummaryKeys); !ok {
				t.Fatal("invalid type for summary keys")
			}
		})
	})

	t.Run("Test Measurer with poisoned DNS: expect TelegramWebFailure", func(t *testing.T) {
		// create a new test environment with bogon DNS
		dnsConfig := netem.NewDNSConfig()
		dnsConfig.AddRecord(
			"web.telegram.org",
			"web.telegram.org", // CNAME
			"a.b.c.d",          // bogon
		)
		env := netemx.NewEnvironment(envConfigWithDNS(dnsConfig))
		defer env.Close()
		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*telegram.TestKeys)
			if tk.TelegramWebFailure == nil {
				t.Fatalf("Expected Web Failure but got none")
			}
			if tk.TelegramHTTPBlocking {
				t.Fatal("Unexpected HTTP blocking")
			}
			if tk.TelegramTCPBlocking {
				t.Fatal("Unexpected TCP blocking")
			}
		})
	})

	t.Run("Test Measurer with DPI that drops TCP traffic towards telegram endpoint: expect Telegram(HTTP|TCP)Blocking", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		// overwrite global Datacenters, otherwise the test times out because there are too many endpoints
		orig := telegram.Datacenters
		telegram.Datacenters = []string{
			"149.154.175.50",
		}
		// create a new test environment
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		// create DPI that drops traffic for datacenter endpoints on ports 443 and 80
		dpi := env.DPIEngine()
		for _, dc := range telegram.Datacenters {
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          model.DiscardLogger,
				ServerIPAddress: dc,
				ServerPort:      80,
				ServerProtocol:  layers.IPProtocolTCP,
			})
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          model.DiscardLogger,
				ServerIPAddress: dc,
				ServerPort:      443,
				ServerProtocol:  layers.IPProtocolTCP,
			})
		}
		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*telegram.TestKeys)
			if tk.TelegramWebFailure != nil {
				t.Fatalf("Unexpected Telegram Web failure %s", *tk.TelegramWebFailure)
			}
			if !tk.TelegramHTTPBlocking {
				t.Fatal("Expected HTTP blocking but got none")
			}
			if !tk.TelegramTCPBlocking {
				t.Fatal("Expected TCP blocking but got none")
			}
		})
		telegram.Datacenters = orig
	})

	t.Run("Test Measurer with DPI that drops TLS traffic with SNI = web.telegram.org: expect TelegramWebFailure", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		// create DPI that drops TLS packets with SNI = web.telegram.org
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "web.telegram.org",
		})
		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*telegram.TestKeys)
			if tk.TelegramWebFailure == nil {
				t.Fatalf("Expected Web Failure but got none")
			}
			if tk.TelegramHTTPBlocking {
				t.Fatal("Unexpected HTTP blocking")
			}
			if tk.TelegramTCPBlocking {
				t.Fatal("Unexpected TCP blocking")
			}
		})
	})
}
