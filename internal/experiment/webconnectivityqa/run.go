package webconnectivityqa

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// RunTestCase runs a [testCase].
func RunTestCase(measurer model.ExperimentMeasurer, tc *TestCase) error {
	// configure the netemx scenario
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()
	if tc.Configure != nil {
		tc.Configure(env)
	}

	// create the measurement skeleton
	t0 := time.Now().UTC()
	measurement := newMeasurement(tc.Input, measurer, t0)

	var err error
	env.Do(func() {
		// create an HTTP client inside the env.Do function so we're using netem
		httpClient := netxlite.NewHTTPClientStdlib(log.Log)
		arguments := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     newSession(httpClient),
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
		return fmt.Errorf("expected to see no error but got %s", err.Error())
	case err == nil && tc.ExpectErr:
		return fmt.Errorf("expected to see an error but got <nil>")
	}

	// reduce the test keys to a common format
	tk := newTestKeys(measurement)

	// compare the expected test keys to the ones we've got
	return compareTestKeys(tc.ExpectTestKeys, tk)
}
