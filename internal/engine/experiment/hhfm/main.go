package hhfm

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ErrNoCheckInInfo indicates check-in returned no suitable info.
var ErrNoCheckInInfo = errors.New("http_header_field_manipulation: returned no check-in info")

// Main is the main function of the experiment.
func Main(ctx context.Context, args *model.ExperimentMainArgs, config *Config) error {
	sess := args.Session
	logger := sess.Logger()

	// Create the directory where to store results unless it already exists
	if err := os.MkdirAll(args.MeasurementDir, 0700); err != nil {
		return err
	}

	// Attempt to remove the results directory when done unless it
	// contains files, in which case we should keep it.
	defer os.Remove(args.MeasurementDir)

	// Call the check-in API to obtain configuration. Note that the value
	// returned here MAY have been cached by the engine.
	logger.Infof("calling check-in API...")
	checkInResp, err := experiment.CallCheckIn(ctx, args, sess)

	// Bail if either the check-in API failed or we don't have a reportID
	// with which to submit Web Connectivity measurements results.
	if err != nil {
		return err
	}
	if checkInResp.HHFM == nil {
		return ErrNoCheckInInfo
	}

	// Obtain and log the report ID.
	reportID := checkInResp.HHFM.ReportID
	logger.Infof("ReportID: %s", reportID)

	// Obtain experiment inputs.
	inputs := getInputs(args, checkInResp)

	// Create an instance of the experiment's measurer.
	measurer := &Measurer{Config: *config}

	// Record when we started running this nettest.
	testStartTime := time.Now()

	// Create suitable stop policy.
	shouldStop := newStopPolicy(args, testStartTime)

	// Create suitable progress emitter.
	progresser := newProgressEmitter(args, inputs, testStartTime)

	// Measure each URL in sequence.
	for inputIdx, input := range inputs {

		// Honour max runtime.
		if shouldStop() {
			break
		}

		// Emit progress.
		progresser(inputIdx, input.URL)

		// Measure the current URL.
		err := experiment.MeasurePossiblyNilInput(
			ctx,
			args,
			measurer,
			testStartTime,
			reportID,
			inputIdx,
			&input,
		)

		// An error here means stuff like "cannot write to disk".
		if err != nil {
			return err
		}
	}

	return nil
}

// getInputs obtains inputs from either args or checkInResp giving
// priority to user supplied arguments inside args.
func getInputs(args *model.ExperimentMainArgs, checkInResp *model.OOAPICheckInNettests) []model.OOAPIURLInfo {
	runtimex.Assert(checkInResp.WebConnectivity != nil, "getInputs passed invalid checkInResp")
	inputs := args.Inputs
	if len(inputs) < 1 {
		return checkInResp.WebConnectivity.URLs
	}
	outputs := []model.OOAPIURLInfo{}
	for _, input := range inputs {
		outputs = append(outputs, model.OOAPIURLInfo{
			CategoryCode: "MISC",
			CountryCode:  "ZZ",
			URL:          input,
		})
	}
	return outputs
}

// newStopPolicy creates a new stop policy depending on the
// arguments passed to the experiment in args.
func newStopPolicy(args *model.ExperimentMainArgs, testStartTime time.Time) func() bool {
	if args.MaxRuntime <= 0 {
		return func() bool {
			return false
		}
	}
	maxRuntime := time.Duration(args.MaxRuntime) * time.Second
	return func() bool {
		return time.Since(testStartTime) > maxRuntime
	}
}

func newProgressEmitter(
	args *model.ExperimentMainArgs,
	inputs []model.OOAPIURLInfo,
	testStartTime time.Time,
) func(idx int, URL string) {
	total := len(inputs)
	if total <= 0 {
		return func(idx int, URL string) {} // just in case
	}
	if args.MaxRuntime <= 0 {
		return func(idx int, URL string) {
			percentage := 100.0 * (float64(idx) / float64(total))
			args.Callbacks.OnProgress(percentage, URL)
		}
	}
	maxRuntime := (time.Duration(args.MaxRuntime) * time.Second) + time.Nanosecond // avoid zero division
	return func(idx int, URL string) {
		elapsed := time.Since(testStartTime)
		percentage := 100.0 * (float64(elapsed) / float64(maxRuntime))
		args.Callbacks.OnProgress(percentage, URL)
	}
}
