package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivityqa"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// newMeasurement constructs a new [model.Measurement].
func newMeasurement(input string, measurer model.ExperimentMeasurer, t0 time.Time) *model.Measurement {
	return &model.Measurement{
		Annotations:               nil,
		DataFormatVersion:         "0.2.0",
		Extensions:                nil,
		ID:                        "",
		Input:                     model.MeasurementTarget(input),
		InputHashes:               nil,
		MeasurementStartTime:      t0.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: t0,
		Options:                   []string{},
		ProbeASN:                  "AS137",
		ProbeCC:                   "IT",
		ProbeCity:                 "",
		ProbeIP:                   "127.0.0.1",
		ProbeNetworkName:          "Consortium GARR",
		ReportID:                  "",
		ResolverASN:               "AS137",
		ResolverIP:                netemx.ISPResolverAddress,
		ResolverNetworkName:       "Consortium GARR",
		SoftwareName:              "ooniprobe",
		SoftwareVersion:           version.Version,
		TestHelpers: map[string]any{
			"backend": map[string]string{
				"address": "https://0.th.ooni.org",
				"type":    "https",
			},
		},
		TestKeys:           nil,
		TestName:           measurer.ExperimentName(),
		MeasurementRuntime: 0,
		TestStartTime:      t0.Format(model.MeasurementDateFormat),
		TestVersion:        measurer.ExperimentVersion(),
	}
}

// newSession creates a new [model.ExperimentSession].
func newSession(client model.HTTPClient, logger model.Logger) model.ExperimentSession {
	return &mocks.Session{
		MockGetTestHelpersByName: func(name string) ([]model.OOAPIService, bool) {
			output := []model.OOAPIService{{
				Address: "https://0.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://1.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://2.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://3.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}}
			return output, true
		},

		MockDefaultHTTPClient: func() model.HTTPClient {
			return client
		},

		MockFetchPsiphonConfig: nil,

		MockFetchTorTargets: nil,

		MockKeyValueStore: nil,

		MockLogger: func() model.Logger {
			return logger
		},

		MockMaybeResolverIP: nil,

		MockProbeASNString: nil,

		MockProbeCC: nil,

		MockProbeIP: nil,

		MockProbeNetworkName: nil,

		MockProxyURL: nil,

		MockResolverIP: func() string {
			return netemx.ISPResolverAddress
		},

		MockSoftwareName: nil,

		MockSoftwareVersion: nil,

		MockTempDir: nil,

		MockTorArgs: nil,

		MockTorBinary: nil,

		MockTunnelDir: nil,

		MockUserAgent: func() string {
			return model.HTTPHeaderUserAgent
		},

		MockNewExperimentBuilder: nil,

		MockNewSubmitter: nil,

		MockCheckIn: nil,
	}
}

func mustSaveAnalysis(destdir string, rawMeasurement []byte) {
	var meas minipipeline.Measurement
	must.UnmarshalJSON(rawMeasurement, &meas)
	container := runtimex.Try1(minipipeline.LoadWebMeasurement(&meas))
	must.WriteFile(
		filepath.Join(destdir, "observations.json"),
		must.MarshalAndIndentJSON(container, "", "  "),
		0600,
	)
	analysis := minipipeline.AnalyzeWebMeasurement(container)
	must.WriteFile(
		filepath.Join(destdir, "analysis.json"),
		must.MarshalAndIndentJSON(analysis, "", "  "),
		0600,
	)
}

func runTestCase(tc *webconnectivityqa.TestCase) {
	measurer := webconnectivitylte.NewExperimentMeasurer(&webconnectivitylte.Config{})

	// configure the netemx scenario
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()
	if tc.Configure != nil {
		tc.Configure(env)
	}

	// create the measurement skeleton
	t0 := time.Now().UTC()
	measurement := newMeasurement(tc.Input, measurer, t0)

	// create a logger for the probe
	prefixLogger := &logx.PrefixLogger{
		Prefix: fmt.Sprintf("%-16s", "PROBE"),
		Logger: log.Log,
	}

	var err error
	env.Do(func() {
		// create an HTTP client inside the env.Do function so we're using netem
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
		httpClient := netxlite.NewHTTPClientStdlib(prefixLogger)
		arguments := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(prefixLogger),
			Measurement: measurement,
			Session:     newSession(httpClient, prefixLogger),
		}

		// run the experiment
		ctx := context.Background()
		err = measurer.Run(ctx, arguments)

		// compute the total measurement runtime
		runtime := time.Since(t0)
		measurement.MeasurementRuntime = runtime.Seconds()
	})

	// handle the case of unexpected result
	runtimex.PanicOnError(err, "measurer.Run failed")

	destdir := filepath.Join(
		"internal", "minipipeline", "testdata", "generated", "webconnectivity",
		tc.Name,
	)
	runtimex.Try0(os.MkdirAll(destdir, 0700))

	rawMeasurement := must.MarshalAndIndentJSON(measurement, "", "  ")
	must.WriteFile(filepath.Join(destdir, "measurement.json"), rawMeasurement, 0600)

	mustSaveAnalysis(destdir, rawMeasurement)
}

func main() {
	for _, tc := range webconnectivityqa.AllTestCases() {
		runTestCase(tc)
	}
}
