package sniblocking

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTestKeysClassify(t *testing.T) {
	asStringPtr := func(s string) *string {
		return &s
	}

	t.Run("with tk.Target.Failure == nil", func(t *testing.T) {
		tk := new(TestKeys)
		if tk.classify() != classSuccessGotServerHello {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == connection_refused", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureConnectionRefused)
		if tk.classify() != classAnomalyTestHelperUnreachable {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == dns_nxdomain_error", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureDNSNXDOMAINError)
		if tk.classify() != classAnomalyTestHelperUnreachable {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == android_dns_cache_no_data", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureAndroidDNSCacheNoData)
		if tk.classify() != classAnomalyTestHelperUnreachable {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == connection_reset", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureConnectionReset)
		if tk.classify() != classInterferenceReset {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == eof_error", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureEOFError)
		if tk.classify() != classInterferenceClosed {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == ssl_invalid_hostname", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureSSLInvalidHostname)
		if tk.classify() != classSuccessGotServerHello {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == ssl_unknown_authority", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureSSLUnknownAuthority)
		if tk.classify() != classInterferenceUnknownAuthority {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == ssl_invalid_certificate", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureSSLInvalidCertificate)
		if tk.classify() != classInterferenceInvalidCertificate {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == generic_timeout_error #1", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureGenericTimeoutError)
		if tk.classify() != classAnomalyTimeout {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == generic_timeout_error #2", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr(netxlite.FailureGenericTimeoutError)
		tk.Control.Failure = asStringPtr(netxlite.FailureGenericTimeoutError)
		if tk.classify() != classAnomalyTestHelperUnreachable {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with tk.Target.Failure == unknown_failure", func(t *testing.T) {
		tk := new(TestKeys)
		tk.Target.Failure = asStringPtr("unknown_failure")
		if tk.classify() != classAnomalyUnexpectedFailure {
			t.Fatal("unexpected result")
		}
	})
}

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "sni_blocking" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected version")
	}
}

func TestProcessallPanicsIfInvalidSNI(t *testing.T) {
	defer func() {
		panicdata := recover()
		if panicdata == nil {
			t.Fatal("expected to see panic here")
		}
		if panicdata.(string) != "unexpected smk.SNI" {
			t.Fatal("not the panic we expected")
		}
	}()
	outputs := make(chan Subresult, 1)
	measurement := &model.Measurement{
		Input: "kernel.org",
	}
	go func() {
		outputs <- Subresult{
			SNI: "antani.io",
		}
	}()
	processall(
		outputs,
		measurement,
		model.NewPrinterCallbacks(model.DiscardLogger),
		[]string{"kernel.org", "example.com"},
		&mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		"example.com",
	)
}

func TestMaybeURLToSNI(t *testing.T) {
	t.Run("for invalid URL", func(t *testing.T) {
		parsed, err := maybeURLToSNI("\t")
		if err == nil {
			t.Fatal("expected an error here")
		}
		if parsed != "" {
			t.Fatal("expected empty parsed here")
		}
	})

	t.Run("for domain name", func(t *testing.T) {
		parsed, err := maybeURLToSNI("kernel.org")
		if err != nil {
			t.Fatal(err)
		}
		if parsed != "kernel.org" {
			t.Fatal("expected different domain here")
		}
	})

	t.Run("for valid URL", func(t *testing.T) {
		parsed, err := maybeURLToSNI("https://kernel.org/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		if parsed != "kernel.org" {
			t.Fatal("expected different domain here")
		}
	})
}

func TestSummaryKeysGeneric(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &TestKeys{}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

// exampleOrgAddr is the IP address used for example.org in netem-based nettests.
const exampleOrgAddr = "93.184.216.34"

// configureDNSWithAddr is like [configureDNSWithDefaults] but uses the given IP addr.
func configureDNSWithAddr(config *netem.DNSConfig, addr string) {
	config.AddRecord("example.org", "example.org", addr)
}

// configureDNSWithDefaults populates the given config using [exampleOrgAddr] as the address.
func configureDNSWithDefaults(config *netem.DNSConfig) {
	configureDNSWithAddr(config, exampleOrgAddr)
}

func TestMeasurerWithInvalidInput(t *testing.T) {
	t.Run("with no measurement input: expect input error", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// we use the same valid DNS config for client and servers here
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			measurer := NewExperimentMeasurer(Config{})
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
				Measurement: &model.Measurement{},
				Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			}
			err := measurer.Run(context.Background(), args)
			if err.Error() != "experiment requires measurement.Input" {
				t.Fatal("not the error we expected")
			}
		})
	})

	t.Run("with invalid MeasurementInput: expect parsing error", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// we use the same valid DNS config for client and servers here
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancel the context
			measurer := NewExperimentMeasurer(Config{
				ControlSNI: "example.org",
			})
			measurement := &model.Measurement{
				Input: "\t",
			}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			}
			err := measurer.Run(ctx, args)
			if err == nil {
				t.Fatal("expected an error here")
			}
		})
	})
}

func TestMeasurerRun(t *testing.T) {
	t.Run("without DPI: expect success", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// we use the same valid DNS config for client and servers here
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			measurer := NewExperimentMeasurer(Config{
				ControlSNI: "example.org",
			})
			measurement := &model.Measurement{
				Input: "kernel.org",
			}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*TestKeys)
			if tk.Result != classSuccessGotServerHello {
				t.Fatalf("Unexpected result, expected: %s, got: %s", classSuccessGotServerHello, tk.Result)
			}
			if tk.Control.Failure != nil {
				t.Fatalf("Unexpected Control Failure %s", *tk.Control.Failure)
			}
			target := tk.Target
			if target.Failure != nil {
				t.Fatalf("Unexpected Target Failure %s", *tk.Target.Failure)
			}
			if target.Agent != "redirect" {
				t.Fatal("not the expected Agent")
			}
			if target.BootstrapTime != 0.0 {
				t.Fatal("not the expected BootstrapTime")
			}
			if target.DNSCache != nil {
				t.Fatal("not the expected DNSCache")
			}
			if target.FailedOperation != nil {
				t.Fatal("unexpected FailedOperation")
			}
			if target.Failure != nil {
				t.Fatal("unexpected failure")
			}
			if len(target.NetworkEvents) < 1 {
				t.Fatal("not the expected NetworkEvents")
			}
			if len(target.Queries) < 1 {
				t.Fatal("not the expected Queries")
			}
			if target.Requests != nil {
				t.Fatal("not the expected Requests")
			}
			if target.SOCKSProxy != "" {
				t.Fatal("not the expected SOCKSProxy")
			}
			if len(target.TCPConnect) < 1 {
				t.Fatal("not the expected TCPConnect")
			}
			if len(target.TLSHandshakes) < 1 {
				t.Fatal("not the expected TLSHandshakes")
			}
			if target.Tunnel != "" {
				t.Fatal("not the expected Tunnel")
			}
			if target.SNI != "kernel.org" {
				t.Fatal("unexpected SNI")
			}
			if target.THAddress != "example.org:443" {
				t.Fatal("unexpected THAddress")
			}
		})
	})

	t.Run("with cancelled context: expect interrupted failure and nil keys", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel the context
		measurer := NewExperimentMeasurer(Config{
			ControlSNI: "example.com",
		})
		measurement := &model.Measurement{
			Input: "kernel.org",
		}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}
		err := measurer.Run(ctx, args)
		if err != nil {
			t.Fatal(err)
		}
		tk := measurement.TestKeys.(*TestKeys)
		if tk.Result != classAnomalyUnexpectedFailure {
			t.Fatalf("Unexpected result, expected: %s, got: %s", classAnomalyUnexpectedFailure, tk.Result)
		}
		sk, err := measurer.GetSummaryKeys(measurement)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := sk.(SummaryKeys); !ok {
			t.Fatal("invalid type for summary keys")
		}
		target := tk.Target
		if target.Agent != "" {
			t.Fatal("not the expected Agent")
		}
		if target.BootstrapTime != 0.0 {
			t.Fatal("not the expected BootstrapTime")
		}
		if target.DNSCache != nil {
			t.Fatal("not the expected DNSCache")
		}
		if target.FailedOperation == nil || *target.FailedOperation != netxlite.TopLevelOperation {
			t.Fatal("not the expected FailedOperation")
		}
		if target.Failure == nil || *target.Failure != netxlite.FailureInterrupted {
			t.Fatal("not the expected failure")
		}
		if target.NetworkEvents != nil {
			t.Fatal("not the expected NetworkEvents")
		}
		if target.Queries != nil {
			t.Fatal("not the expected Queries")
		}
		if target.Requests != nil {
			t.Fatal("not the expected Requests")
		}
		if target.SOCKSProxy != "" {
			t.Fatal("not the expected SOCKSProxy")
		}
		if target.TCPConnect != nil {
			t.Fatal("not the expected TCPConnect")
		}
		if target.TLSHandshakes != nil {
			t.Fatal("not the expected TLSHandshakes")
		}
		if target.Tunnel != "" {
			t.Fatal("not the expected Tunnel")
		}
		if target.SNI != "kernel.org" {
			t.Fatal("unexpected SNI")
		}
		if target.THAddress != "example.com:443" {
			t.Fatal("unexpected THAddress")
		}
	})

	t.Run("with cache: expect to see cached entry", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// we use the same valid DNS config for client and servers here
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			cache := make(map[string]Subresult)
			s := "mock error"
			testsni := "kernel.org"
			thaddr := "example.org:443"
			subresult := Subresult{
				Cached:    true,
				THAddress: thaddr,
				SNI:       testsni,
			}
			subresult.Failure = &s
			cache[testsni+thaddr] = subresult
			measurer := NewExperimentMeasurer(Config{
				ControlSNI: "example.org",
			})
			measurer.(*Measurer).cache = cache
			measurement := &model.Measurement{
				Input: model.MeasurementTarget(testsni),
			}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*TestKeys)
			if tk.Result != classAnomalyUnexpectedFailure {
				t.Fatalf("Unexpected result, expected: %s, got: %s", classAnomalyUnexpectedFailure, tk.Result)
			}
			if tk.Control.Failure != nil {
				t.Fatalf("Unexpected Control Failure %s", *tk.Control.Failure)
			}
			if tk.Target.Failure == nil {
				t.Fatalf("Expected Target Failure but got none")
			}
			if *tk.Target.Failure != "mock error" {
				t.Fatalf("Unexpected Target Failure, expected: %s, got: %s", "mock error", *tk.Target.Failure)
			}
			if !tk.Target.Cached {
				t.Fatalf("Expected Cached = true")
			}
		})
	})

	t.Run("with DPI that blocks target SNI", func(t *testing.T) {
		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// we use the same valid DNS config for client and servers here
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "kernel.org",
		})

		env.Do(func() {
			measurer := NewExperimentMeasurer(Config{
				ControlSNI: "example.org",
			})
			measurement := &model.Measurement{
				Input: "kernel.org",
			}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
				Measurement: measurement,
				Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*TestKeys)
			if tk.Result != classInterferenceReset {
				t.Fatalf("Unexpected result, expected: %s, got: %s", classInterferenceReset, tk.Result)
			}
			if tk.Control.Failure != nil {
				t.Fatalf("Unexpected Control Failure %s", *tk.Control.Failure)
			}
			if tk.Target.Failure == nil {
				t.Fatalf("Expected a Target Failure, but got none")
			}
			if *tk.Target.Failure != netxlite.FailureConnectionReset {
				t.Fatalf("Unexpected Target Failure, got: %s, expected: %s", *tk.Target.Failure, netxlite.FailureConnectionReset)
			}
		})
	})
}

func TestMeasureonewithcacheWorks(t *testing.T) {
	// create a new test environment
	env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(exampleOrgAddr, netemx.QAEnvDefaultHTTPHandler()))
	defer env.Close()

	// we use the same valid DNS config for client and servers here
	configureDNSWithDefaults(env.ISPResolverConfig())
	configureDNSWithDefaults(env.OtherResolversConfig())

	env.Do(func() {
		measurer := &Measurer{cache: make(map[string]Subresult)}
		output := make(chan Subresult, 2)
		for i := 0; i < 2; i++ {
			measurer.measureonewithcache(
				context.Background(),
				output,
				&mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
				time.Now(),
				"kernel.org",
				"example.org:443",
			)
		}
		for _, expected := range []bool{false, true} {
			result := <-output
			if result.Cached != expected {
				t.Fatal("unexpected cached")
			}
			if result.Failure != nil {
				t.Fatal("unexpected failure")
			}
			if result.SNI != "kernel.org" {
				t.Fatal("unexpected SNI")
			}
		}
	})
}
