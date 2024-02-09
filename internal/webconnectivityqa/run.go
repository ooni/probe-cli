package webconnectivityqa

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// MeasureTestCase returns the JSON measurement produced by a [TestCase].
func MeasureTestCase(measurer model.ExperimentMeasurer, tc *TestCase) (*model.Measurement, error) {
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
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS
		// but they're not needed here
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
	switch {
	case err != nil && !tc.ExpectErr:
		return nil, fmt.Errorf("expected to see no error but got %s", err.Error())
	case err == nil && tc.ExpectErr:
		return nil, fmt.Errorf("expected to see an error but got <nil>")
	}

	return measurement, nil
}

// RunTestCase runs a [testCase].
func RunTestCase(measurer model.ExperimentMeasurer, tc *TestCase) error {
	// run the test case proper to get a full OONI measurement
	measurement, err := MeasureTestCase(measurer, tc)
	if err != nil {
		return err
	}

	// run each check in the list of checkers
	for _, checker := range tc.Checkers {
		if err := checker.Check(measurement); err != nil {
			return err
		}
	}

	// reduce the test keys to a common format
	tk := newTestKeys(measurement)

	// compare the expected test keys to the ones we've got
	return compareTestKeys(tc.ExpectTestKeys, tk)
}
