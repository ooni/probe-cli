package sniblocking

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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
		model.NewPrinterCallbacks(log.Log),
		[]string{"kernel.org", "example.com"},
		newsession(),
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

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
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

// The netemx environment design is based on netemx_test.

// Environment is the [netem] QA environment we use in this package.
type Environment struct {
	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// dnsServer is the DNS server.
	dnsServer *netem.DNSServer

	// dpi refers to the [netem.DPIEngine] we're using
	dpi *netem.DPIEngine

	// httpsServer is the HTTPS server.
	httpsServer *http.Server

	// topology is the topology we're using
	topology *netem.StarTopology
}

// NewEnvironment creates a new QA environment. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewEnvironment(altResolver string) *Environment {
	// create a new star topology
	topology := runtimex.Try1(netem.NewStarTopology(model.DiscardLogger))
	defaultResolver := "1.2.3.4"
	resolverAddr := defaultResolver
	if altResolver != "" {
		resolverAddr = altResolver
	}

	// create server stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	dnsServerStack := runtimex.Try1(topology.AddHost(
		resolverAddr,    // server IP address
		defaultResolver, // default resolver address
		&netem.LinkConfig{},
	))

	// create configuration for DNS server
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord(
		"example.org",
		"example.org", // CNAME
		"9.9.9.9",
	)

	// create DNS server using the dnsServerStack
	dnsServer := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		dnsServerStack,
		resolverAddr,
		dnsConfig,
	))

	serverStack := runtimex.Try1(topology.AddHost(
		"9.9.9.9",       // server IP address
		defaultResolver, // default resolver address
		&netem.LinkConfig{},
	))

	// create HTTPS server using the server stack
	tlsListener := runtimex.Try1(serverStack.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(9, 9, 9, 9),
		Port: 443,
		Zone: "",
	}))
	httpsServer := &http.Server{
		TLSConfig: serverStack.ServerTLSConfig(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`hello, world`))
		}),
	}
	go httpsServer.ServeTLS(tlsListener, "", "")

	// create a DPIEngine for implementing censorship
	dpi := netem.NewDPIEngine(model.DiscardLogger)

	// create client stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	clientStack := runtimex.Try1(topology.AddHost(
		"10.0.0.14",     // client IP address
		defaultResolver, // default resolver address
		&netem.LinkConfig{
			DPIEngine: dpi,
		},
	))

	return &Environment{
		clientStack: clientStack,
		dnsServer:   dnsServer,
		dpi:         dpi,
		httpsServer: httpsServer,
		topology:    topology,
	}
}

// DPIEngine returns the [netem.DPIEngine] we're using on the
// link between the client stack and the router. You can safely
// add new DPI rules from concurrent goroutines at any time.
func (e *Environment) DPIEngine() *netem.DPIEngine {
	return e.dpi
}

// Do executes the given function such that [netxlite] code uses the
// underlying clientStack rather than ordinary networking code.
func (e *Environment) Do(function func()) {
	netemx.WithCustomTProxy(e.clientStack, function)
}

// Close closes all the resources used by [Environment].
func (e *Environment) Close() error {
	e.dnsServer.Close()
	e.httpsServer.Close()
	e.topology.Close()
	return nil
}

func TestMeasurerWithInvalidInput(t *testing.T) {
	t.Run("Test Measurer with no measurement input: expect input error", func(t *testing.T) {
		env := NewEnvironment("")
		defer env.Close()
		env.Do(func() {
			measurer := NewExperimentMeasurer(Config{})
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: &model.Measurement{},
				Session:     newsession(),
			}
			err := measurer.Run(context.Background(), args)
			if err.Error() != "Experiment requires measurement.Input" {
				t.Fatal("not the error we expected")
			}
		})
	})
	t.Run("Test Measurer with invalid MeasurementInput: expect parsing error", func(t *testing.T) {
		env := NewEnvironment("")
		defer env.Close()
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
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
			}
			err := measurer.Run(ctx, args)
			if err == nil {
				t.Fatal("expected an error here")
			}
		})
	})

}
func TestMeasurerRun(t *testing.T) {

	t.Run("Test Measurer without DPI: expect success", func(t *testing.T) {
		env := NewEnvironment("")
		defer env.Close()
		env.Do(func() {
			measurer := NewExperimentMeasurer(Config{
				ControlSNI: "example.org",
			})
			measurement := &model.Measurement{
				Input: "kernel.org",
			}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
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

	t.Run("Test Measurer with cancelled context: expect interrupted failure and nil keys", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel the context
		measurer := NewExperimentMeasurer(Config{
			ControlSNI: "example.com",
		})
		measurement := &model.Measurement{
			Input: "kernel.org",
		}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(log.Log),
			Measurement: measurement,
			Session:     newsession(),
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

	t.Run("Test Measurer with cache: expect to see cached entry", func(t *testing.T) {
		env := NewEnvironment("")
		defer env.Close()
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
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
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

	t.Run("Test Measurer with DPI that blocks target SNI", func(t *testing.T) {
		env := NewEnvironment("")
		defer env.Close()
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
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     newsession(),
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
	env := NewEnvironment("")
	defer env.Close()
	env.Do(func() {
		measurer := &Measurer{cache: make(map[string]Subresult)}
		output := make(chan Subresult, 2)
		for i := 0; i < 2; i++ {
			measurer.measureonewithcache(
				context.Background(),
				output,
				&mockable.Session{MockableLogger: log.Log},
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
