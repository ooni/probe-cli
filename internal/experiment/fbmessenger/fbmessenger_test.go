package fbmessenger_test

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// servicesAddr is the IP address implementing al fbmessenger services in netem-based tests
const servicesAddr = "157.240.20.35"

// configureDNSWithAddr is like [configureDNS] but uses a specific addr.
func configureDNSWithAddr(config *netem.DNSConfig, addr string) {
	for _, svc := range fbmessenger.Services {
		hostname := runtimex.Try1(url.Parse(svc)).Hostname()
		config.AddRecord(hostname, hostname, addr)
	}
}

// configureDNSWithDefaults configures the given [*netem.DNSConfig] with all the required domains
func configureDNSWithDefaults(config *netem.DNSConfig) {
	configureDNSWithAddr(config, servicesAddr)
}

var (
	trueValue  = true
	falseValue = false
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
	if measurer.ExperimentName() != "facebook_messenger" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.1" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerRun(t *testing.T) {
	t.Run("without DPI: expect success", func(t *testing.T) {
		// we expect the following Analysis values
		expectAnalysis := fbmessenger.Analysis{
			FacebookBAPIDNSConsistent:        &trueValue,
			FacebookBAPIReachable:            &trueValue,
			FacebookBGraphDNSConsistent:      &trueValue,
			FacebookBGraphReachable:          &trueValue,
			FacebookEdgeDNSConsistent:        &trueValue,
			FacebookEdgeReachable:            &trueValue,
			FacebookExternalCDNDNSConsistent: &trueValue,
			FacebookExternalCDNReachable:     &trueValue,
			FacebookScontentCDNDNSConsistent: &trueValue,
			FacebookScontentCDNReachable:     &trueValue,
			FacebookStarDNSConsistent:        &trueValue,
			FacebookStarReachable:            &trueValue,
			FacebookSTUNDNSConsistent:        &trueValue,
			FacebookSTUNReachable:            nil,
			FacebookDNSBlocking:              &falseValue,
			FacebookTCPBlocking:              &falseValue,
		}

		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(servicesAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// configure the DNS for all resolvers
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			measurement := new(model.Measurement)
			callbacks := model.NewPrinterCallbacks(log.Log)
			args := &model.ExperimentArgs{
				Callbacks:   callbacks,
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(ctx, args)
			if err != nil {
				t.Fatal(err)
			}
			tk := measurement.TestKeys.(*fbmessenger.TestKeys)
			if diff := cmp.Diff(expectAnalysis, tk.Analysis); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("without DPI, cancelled context: expect interrupted failure", func(t *testing.T) {
		// we expect the following Analysis values
		expectAnalysis := fbmessenger.Analysis{
			FacebookBAPIDNSConsistent:        &falseValue,
			FacebookBAPIReachable:            nil,
			FacebookBGraphDNSConsistent:      &falseValue,
			FacebookBGraphReachable:          nil,
			FacebookEdgeDNSConsistent:        &falseValue,
			FacebookEdgeReachable:            nil,
			FacebookExternalCDNDNSConsistent: &falseValue,
			FacebookExternalCDNReachable:     nil,
			FacebookScontentCDNDNSConsistent: &falseValue,
			FacebookScontentCDNReachable:     nil,
			FacebookStarDNSConsistent:        &falseValue,
			FacebookStarReachable:            nil,
			FacebookSTUNDNSConsistent:        &falseValue,
			FacebookSTUNReachable:            nil,
			FacebookDNSBlocking:              &trueValue,
			FacebookTCPBlocking:              &falseValue, // no TCP blocking because we didn't ever reach TCP connect
		}

		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(servicesAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// configure the DNS for all resolvers
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // so we fail immediately
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			measurement := new(model.Measurement)
			callbacks := model.NewPrinterCallbacks(log.Log)
			args := &model.ExperimentArgs{
				Callbacks:   callbacks,
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(ctx, args)
			if err != nil {
				t.Fatal(err)
			}
			tk := measurement.TestKeys.(*fbmessenger.TestKeys)
			if diff := cmp.Diff(expectAnalysis, tk.Analysis); diff != "" {
				t.Fatal(diff)
			}
			sk, err := measurer.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := sk.(fbmessenger.SummaryKeys); !ok {
				t.Fatal("invalid type for summary keys")
			}
		})
	})

	t.Run("Test Measurer with DPI that drops traffic to fbmessenger endpoint: expect FacebookTCPBlocking", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// we expect the following Analysis values
		expectAnalysis := fbmessenger.Analysis{
			FacebookBAPIDNSConsistent: &trueValue,
			FacebookBAPIReachable:     &falseValue,
			FacebookDNSBlocking:       &falseValue,
			FacebookTCPBlocking:       &trueValue,
		}

		// overwrite global Services, otherwise the test times out because there are too many endpoints
		orig := fbmessenger.Services
		fbmessenger.Services = []string{
			fbmessenger.ServiceBAPI,
		}
		defer func() {
			fbmessenger.Services = orig
		}()

		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(servicesAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// configure the DNS for all resolvers
		configureDNSWithDefaults(env.ISPResolverConfig())
		configureDNSWithDefaults(env.OtherResolversConfig())

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
			Logger:          model.DiscardLogger,
			ServerIPAddress: servicesAddr,
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		})

		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			measurement := new(model.Measurement)
			callbacks := model.NewPrinterCallbacks(log.Log)
			args := &model.ExperimentArgs{
				Callbacks:   callbacks,
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(ctx, args)
			if err != nil {
				t.Fatal(err)
			}
			tk := measurement.TestKeys.(*fbmessenger.TestKeys)
			if diff := cmp.Diff(expectAnalysis, tk.Analysis); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("with poisoned DNS: expect FacebookDNSBlocking", func(t *testing.T) {
		// we expect the following Analysis values
		expectAnalysis := fbmessenger.Analysis{
			FacebookBAPIDNSConsistent:        &falseValue,
			FacebookBAPIReachable:            nil,
			FacebookBGraphDNSConsistent:      &falseValue,
			FacebookBGraphReachable:          nil,
			FacebookEdgeDNSConsistent:        &falseValue,
			FacebookEdgeReachable:            nil,
			FacebookExternalCDNDNSConsistent: &falseValue,
			FacebookExternalCDNReachable:     nil,
			FacebookScontentCDNDNSConsistent: &falseValue,
			FacebookScontentCDNReachable:     nil,
			FacebookStarDNSConsistent:        &falseValue,
			FacebookStarReachable:            nil,
			FacebookSTUNDNSConsistent:        &falseValue,
			FacebookSTUNReachable:            nil,
			FacebookDNSBlocking:              &trueValue,
			FacebookTCPBlocking:              &falseValue, // no TCP blocking because we didn't ever reach TCP connect
		}

		// create a new test environment
		env := netemx.NewQAEnv(netemx.QAEnvOptionHTTPServer(servicesAddr, netemx.QAEnvDefaultHTTPHandler()))
		defer env.Close()

		// configure all DNS servers but the ISP's one
		configureDNSWithDefaults(env.OtherResolversConfig())

		// configure the ISP resolver to use bogons
		configureDNSWithAddr(env.ISPResolverConfig(), "10.10.34.35")

		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			measurement := new(model.Measurement)
			callbacks := model.NewPrinterCallbacks(log.Log)
			args := &model.ExperimentArgs{
				Callbacks:   callbacks,
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(ctx, args)
			if err != nil {
				t.Fatal(err)
			}
			tk := measurement.TestKeys.(*fbmessenger.TestKeys)
			if diff := cmp.Diff(expectAnalysis, tk.Analysis); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

func TestComputeEndpointStatsTCPBlocking(t *testing.T) {
	failure := io.EOF.Error()
	operation := netxlite.ConnectOperation
	tk := fbmessenger.TestKeys{}
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{Target: fbmessenger.ServiceEdge},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &operation,
			Queries: []tracex.DNSQueryEntry{{
				Answers: []tracex.DNSAnswerEntry{{
					ASN: fbmessenger.FacebookASN,
				}},
			}},
		},
	})
	if *tk.FacebookEdgeDNSConsistent != true {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if *tk.FacebookEdgeReachable != false {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if tk.FacebookDNSBlocking != nil { // meaning: not determined yet
		t.Fatal("invalid FacebookDNSBlocking")
	}
	if *tk.FacebookTCPBlocking != true {
		t.Fatal("invalid FacebookTCPBlocking")
	}
}

func TestComputeEndpointStatsDNSIsLying(t *testing.T) {
	failure := io.EOF.Error()
	operation := netxlite.ConnectOperation
	tk := fbmessenger.TestKeys{}
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{Target: fbmessenger.ServiceEdge},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &operation,
			Queries: []tracex.DNSQueryEntry{{
				Answers: []tracex.DNSAnswerEntry{{
					ASN: 0,
				}},
			}},
		},
	})
	if *tk.FacebookEdgeDNSConsistent != false {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if tk.FacebookEdgeReachable != nil {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if *tk.FacebookDNSBlocking != true {
		t.Fatal("invalid FacebookDNSBlocking")
	}
	if tk.FacebookTCPBlocking != nil { // meaning: not determined yet
		t.Fatal("invalid FacebookTCPBlocking")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &fbmessenger.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWithNils(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithFalseFalse(t *testing.T) {
	falsy := false
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		Analysis: fbmessenger.Analysis{
			FacebookTCPBlocking: &falsy,
			FacebookDNSBlocking: &falsy,
		},
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithFalseTrue(t *testing.T) {
	falsy := false
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		Analysis: fbmessenger.Analysis{
			FacebookTCPBlocking: &falsy,
			FacebookDNSBlocking: &truy,
		},
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking == false {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithTrueFalse(t *testing.T) {
	falsy := false
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		Analysis: fbmessenger.Analysis{
			FacebookTCPBlocking: &truy,
			FacebookDNSBlocking: &falsy,
		},
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking == false {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithTrueTrue(t *testing.T) {
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		Analysis: fbmessenger.Analysis{
			FacebookTCPBlocking: &truy,
			FacebookDNSBlocking: &truy,
		},
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking == false {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking == false {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}
