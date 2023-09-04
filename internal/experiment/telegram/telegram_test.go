package telegram_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := telegram.NewExperimentMeasurer(telegram.Config{})
	if measurer.ExperimentName() != "telegram" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.1" {
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

// telegramWebAddr is the web.telegram.org IP address as of 2023-07-11
const telegramWebAddr = "149.154.167.99"

// configureDNSWithAddr configures the given DNS config for web.telegram.org using the given addr.
func configureDNSWithAddr(config *netem.DNSConfig, addr string) {
	config.AddRecord("web.telegram.org", "web.telegram.org", addr)
}

// configureDNSWithDefaults configures the given DNS config for web.telegram.org using the default addr.
func configureDNSWithDefaults(config *netem.DNSConfig) {
	configureDNSWithAddr(config, telegramWebAddr)
}

// newQAEnvironment creates a QA environment for testing using the given addresses.
func newQAEnvironment(ipaddrs ...string) *netemx.QAEnv {
	// create a single factory for handling all the requests
	factory := &netemx.HTTPCleartextServerFactory{
		Factory: netemx.HTTPHandlerFactoryFunc(func() http.Handler {
			// we create an empty mux, which should cause a 404 for each webpage, which seems what
			// the servers used by telegram DC do as of 2023-07-11
			return http.NewServeMux()
		}),
		Ports: []int{80, 443},
	}

	// create the options for constructing the env
	var options []netemx.QAEnvOption
	for _, ipaddr := range ipaddrs {
		options = append(options, netemx.QAEnvOptionNetStack(ipaddr, factory))
	}

	// add explicit logging which helps to inspect the tests results
	options = append(options, netemx.QAEnvOptionLogger(log.Log))

	// add handler for telegram web (we're using a different-from-reality HTTP handler
	// but we're not testing for the returned webpage, so we should be fine)
	options = append(options, netemx.QAEnvOptionHTTPServer(telegramWebAddr, netemx.ExampleWebPageHandlerFactory()))

	// create the environment proper with all the options
	env := netemx.MustNewQAEnv(options...)

	// register with all the possible resolvers the correct DNS records - registering again
	// inside individual tests will override the values we're setting here
	configureDNSWithDefaults(env.ISPResolverConfig())
	configureDNSWithDefaults(env.OtherResolversConfig())
	return env
}

func TestMeasurerRun(t *testing.T) {
	t.Run("without DPI: expect success", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment(telegram.DatacenterIPAddrs...)
		defer env.Close()

		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return log.Log }},
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

	t.Run("with poisoned DNS: expect TelegramWebFailure", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment(telegram.DatacenterIPAddrs...)
		defer env.Close()

		// register bogon entries for web.telegram.org in the resolver's ISP
		env.ISPResolverConfig().AddRecord("web.telegram.org", "web.telegram.org", "10.10.34.35")

		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return log.Log }},
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

	t.Run("with DPI that drops TCP traffic towards telegram endpoint: expect Telegram(HTTP|TCP)Blocking", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// overwrite global Datacenters, otherwise the test times out because there are too many endpoints
		orig := telegram.DatacenterIPAddrs
		telegram.DatacenterIPAddrs = []string{
			"149.154.175.50",
		}
		defer func() {
			telegram.DatacenterIPAddrs = orig
		}()

		// create a new test environment
		env := newQAEnvironment(telegram.DatacenterIPAddrs...)
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		for _, dc := range telegram.DatacenterIPAddrs {
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: dc,
				ServerPort:      80,
				ServerProtocol:  layers.IPProtocolTCP,
			})
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          log.Log,
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
				Session:     &mocks.Session{MockLogger: func() model.Logger { return log.Log }},
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
	})

	t.Run("with DPI that drops TLS traffic with SNI = web.telegram.org: expect TelegramWebFailure", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment(telegram.DatacenterIPAddrs...)
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: log.Log,
			SNI:    "web.telegram.org",
		})

		env.Do(func() {
			measurer := telegram.NewExperimentMeasurer(telegram.Config{})
			measurement := &model.Measurement{}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return log.Log }},
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
