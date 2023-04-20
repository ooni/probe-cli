package fbmessenger_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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
		// we expect the following TestKeys values
		expectTestKeys := &fbmessenger.TestKeys{
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
		env := NewEnvironment()
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
			if !tk.EqualResults(expectTestKeys) {
				t.Fatal("invalid TestKeys")
			}
		})
	})
	t.Run("Test Measurer without DPI, cancelled context: expect interrupted failure", func(t *testing.T) {
		// we expect the following TestKeys values
		expectTestKeys := &fbmessenger.TestKeys{
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
		env := NewEnvironment()
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
			if !tk.EqualResults(expectTestKeys) {
				t.Fatal("invalid TestKeys")
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
		// we expect the following TestKeys
		expectTestKeys := &fbmessenger.TestKeys{
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
		env := NewEnvironment()
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
			if !tk.EqualResults(expectTestKeys) {
				t.Fatal("invalid TestKeys")
			}
		})
		fbmessenger.Services = orig
	})
	t.Run("Test Measurer with poisoned DNS: expect FacebookDNSBlocking", func(t *testing.T) {
		// we expect the following TestKeys values
		expectTestKeys := &fbmessenger.TestKeys{
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
				s,         // CNAME
				"a.b.c.d", //bogon
			)
		}
		env := NewEnvironmentWithDNSConfig(dnsConfig)
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
			if !tk.EqualResults(expectTestKeys) {
				t.Fatal("invalid TestKeys")
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
		FacebookTCPBlocking: &falsy,
		FacebookDNSBlocking: &falsy,
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
		FacebookTCPBlocking: &falsy,
		FacebookDNSBlocking: &truy,
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
		FacebookTCPBlocking: &truy,
		FacebookDNSBlocking: &falsy,
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
		FacebookTCPBlocking: &truy,
		FacebookDNSBlocking: &truy,
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

// The netemx environment design is based on netemx_test.

// Environment is the [netem] QA environment we use in this package.
type Environment struct {
	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// dnsServer is the DNS server.
	dnsServer *netem.DNSServer

	// dpi refers to the [netem.DPIEngine] we're using
	dpi *netem.DPIEngine

	// httpServer is the HTTP server.
	httpServer *http.Server

	// topology is the topology we're using
	topology *netem.StarTopology
}

func NewEnvironment() *Environment {
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
	return NewEnvironmentWithDNSConfig(dnsConfig)
}

// NewEnvironment creates a new QA environment. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewEnvironmentWithDNSConfig(dnsConfig *netem.DNSConfig) *Environment {
	e := &Environment{}

	// create a new star topology
	e.topology = runtimex.Try1(netem.NewStarTopology(model.DiscardLogger))

	// create server stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	dnsServerStack := runtimex.Try1(e.topology.AddHost(
		"1.1.1.1", // server IP address
		"0.0.0.0", // default resolver address
		&netem.LinkConfig{},
	))

	// create DNS server using the dnsServerStack
	e.dnsServer = runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		dnsServerStack,
		"1.1.1.1",
		dnsConfig,
	))

	// create a server stack
	httpServerStack := runtimex.Try1(e.topology.AddHost(
		"157.240.20.35", // server IP address
		"0.0.0.0",       // default resolver address
		&netem.LinkConfig{},
	))

	// create a TCP server on port 443
	tcpListener := runtimex.Try1(httpServerStack.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(157, 240, 20, 35),
		Port: 443,
		Zone: "",
	}))
	e.httpServer = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`hello, world`))
		}),
	}
	// run TCP server
	go e.httpServer.Serve(tcpListener)

	// create a DPIEngine for implementing censorship
	e.dpi = netem.NewDPIEngine(model.DiscardLogger)

	// create client stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	e.clientStack = runtimex.Try1(e.topology.AddHost(
		"10.0.0.14", // client IP address
		"1.1.1.1",   // default resolver address
		&netem.LinkConfig{
			DPIEngine: e.dpi,
		},
	))

	return e
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
	e.httpServer.Close()
	e.topology.Close()
	return nil
}
