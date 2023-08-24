package webconnectivitylte

import (
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivityqa"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

/*
func TestSuccess(t *testing.T) {
	env := newEnvironment()
	defer env.Close()
	env.Do(func() {
		measurer := NewExperimentMeasurer(&Config{})
		ctx := context.Background()
		sess := newSession()
		measurement := &model.Measurement{Input: "https://www.example.com"}
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
		tk := measurement.TestKeys.(*TestKeys)
		if tk.ControlFailure != nil {
			t.Fatal("unexpected control_failure", *tk.ControlFailure)
		}
		if tk.Blocking != false {
			t.Fatal("unexpected blocking detected")
		}
		if tk.Accessible != true {
			t.Fatal("unexpected accessible flag: should be accessible")
		}
	})
}

func TestDPITarget(t *testing.T) {
	env := newEnvironment()
	dpi := env.DPIEngine()
	dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
		Logger: model.DiscardLogger,
		SNI:    "www.example.com",
	})
	defer env.Close()
	env.Do(func() {
		measurer := NewExperimentMeasurer(&Config{})
		ctx := context.Background()
		sess := newSession()
		measurement := &model.Measurement{Input: "https://www.example.com"}
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
		tk := measurement.TestKeys.(*TestKeys)
		if tk.ControlFailure != nil {
			t.Fatal("unexpected control_failure", *tk.ControlFailure)
		}
		if tk.Blocking != "http-failure" {
			t.Fatal("unexpected blocking type")
		}
		if tk.Accessible == true {
			t.Fatal("unexpected accessible flag: should be false")
		}
	})
}

// newSession creates a new [mocks.Session].
func newSession() model.ExperimentSession {
	byteCounter := bytecounter.New()
	resolver := &sessionresolver.Resolver{
		ByteCounter: byteCounter,
		KVStore:     &kvstore.Memory{},
		Logger:      log.Log,
		ProxyURL:    nil,
	}
	txp := netxlite.NewHTTPTransportWithLoggerResolverAndOptionalProxyURL(
		log.Log, resolver, nil,
	)
	txp = bytecounter.WrapHTTPTransport(txp, byteCounter)
	return &mocks.Session{
		MockGetTestHelpersByName: func(name string) ([]model.OOAPIService, bool) {
			output := []model.OOAPIService{
				{
					Address: "https://3.th.ooni.org",
					Type:    "https",
				},
				{
					Address: "https://2.th.ooni.org",
					Type:    "https",
				},
				{
					Address: "https://1.th.ooni.org",
					Type:    "https",
				},
				{
					Address: "https://0.th.ooni.org",
					Type:    "https",
				},
			}
			return output, true
		},
		MockDefaultHTTPClient: func() model.HTTPClient {
			return &http.Client{Transport: txp}
		},
		MockFetchPsiphonConfig: nil,
		MockFetchTorTargets:    nil,
		MockKeyValueStore:      nil,
		MockLogger: func() model.Logger {
			return log.Log
		},
		MockMaybeResolverIP:  nil,
		MockProbeASNString:   nil,
		MockProbeCC:          nil,
		MockProbeIP:          nil,
		MockProbeNetworkName: nil,
		MockProxyURL:         nil,
		MockResolverIP:       nil,
		MockSoftwareName:     nil,
		MockSoftwareVersion:  nil,
		MockTempDir:          nil,
		MockTorArgs:          nil,
		MockTorBinary:        nil,
		MockTunnelDir:        nil,
		MockUserAgent: func() string {
			return model.HTTPHeaderUserAgent
		},
		MockNewExperimentBuilder: nil,
		MockNewSubmitter:         nil,
		MockCheckIn:              nil,
	}
}
*/

func TestWebConnectivityQA(t *testing.T) {
	for _, tc := range webconnectivityqa.AllTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			measurer := NewExperimentMeasurer(&Config{
				DNSOverUDPResolver: net.JoinHostPort(netemx.QAEnvDefaultUncensoredResolverAddress, "53"),
			})
			if err := webconnectivityqa.RunTestCase(measurer, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
