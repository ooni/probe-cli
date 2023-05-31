package fbmessenger_test

import (
	"context"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

var (
	trueValue  = true
	falseValue = false
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
	if measurer.ExperimentName() != "facebook_messenger" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerRun(t *testing.T) {
	t.Run("Test Measurer without DPI: expect success", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
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
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			// we need a real session because we need the ASN database
			sess := newsession()
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

	t.Run("Test Measurer without DPI, cancelled context: expect interrupted failure", func(t *testing.T) {
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
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // so we fail immediately
			sess := &mockable.Session{MockableLogger: log.Log}
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
		env := netemx.NewEnvironment(envConfig())
		defer env.Close()
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
			Logger:          model.DiscardLogger,
			ServerIPAddress: "157.240.20.35",
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		})

		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			// we need a real session because we need the ASN database
			sess := newsession()
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
		fbmessenger.Services = orig
	})
	t.Run("Test Measurer with poisoned DNS: expect FacebookDNSBlocking", func(t *testing.T) {
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

		// create a new test environment with bogon DNS
		dnsConfig := netem.NewDNSConfig()
		services := []string{
			"stun.fbsbx.com",
			"b-api.facebook.com",
			"b-graph.facebook.com",
			"edge-mqtt.facebook.com",
			"external.xx.fbcdn.net",
			"scontent.xx.fbcdn.net",
			"star.c10r.facebook.com",
		}
		for _, s := range services {
			// create configuration for DNS server
			dnsConfig.AddRecord(
				s,
				s,             // CNAME
				"10.10.34.35", //bogon
			)
		}
		env := netemx.NewEnvironment(envConfigWithDNS(dnsConfig))
		defer env.Close()
		env.Do(func() {
			measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
			ctx := context.Background()
			// we need a real session because we need the ASN database
			sess := newsession()
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

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
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

// Creates an experiment-specific configuration for the [netemx.Environment].
func envConfig() netemx.Config {
	dnsConfig := netem.NewDNSConfig()
	services := []string{
		"stun.fbsbx.com",
		"b-api.facebook.com",
		"b-graph.facebook.com",
		"edge-mqtt.facebook.com",
		"external.xx.fbcdn.net",
		"scontent.xx.fbcdn.net",
		"star.c10r.facebook.com",
	}
	for _, s := range services {
		// create configuration for DNS server
		dnsConfig.AddRecord(
			s,
			s, // CNAME
			"157.240.20.35",
		)
	}
	return envConfigWithDNS(dnsConfig)
}

// Creates an experiment-specific configuration for the [netemx.Environment]
// with custom DNS.
func envConfigWithDNS(dnsConfig *netem.DNSConfig) netemx.Config {
	return netemx.Config{
		DNSConfig: dnsConfig,
		Servers: []netemx.ServerStack{
			{
				ServerAddr: "157.240.20.35",
				HTTPServers: []netemx.HTTPServer{
					{
						Port: 443,
					},
				},
			},
		},
	}
}
