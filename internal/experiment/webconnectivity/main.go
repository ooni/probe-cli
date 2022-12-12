package webconnectivity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ErrNoCheckInInfo indicates check-in returned no suitable info.
var ErrNoCheckInInfo = errors.New("webconnectivity: returned no check-in info")

// Main is the main function of the experiment.
func Main(
	ctx context.Context,
	args *model.ExperimentMainArgs,
) error {
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
	checkInResp, err := sess.CheckIn(ctx, &model.OOAPICheckInConfig{
		Charging:        args.Charging,
		OnWiFi:          args.OnWiFi,
		Platform:        sess.Platform(),
		ProbeASN:        sess.ProbeASNString(),
		ProbeCC:         sess.ProbeCC(),
		RunType:         args.RunType,
		SoftwareName:    sess.SoftwareName(),
		SoftwareVersion: sess.SoftwareVersion(),
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: args.CategoryCodes,
		},
	})

	// Bail if either the check-in API failed or we don't have a reportID
	// with which to submit Web Connectivity measurements results.
	if err != nil {
		return err
	}
	if checkInResp.WebConnectivity == nil {
		return ErrNoCheckInInfo
	}

	// Obtain and log the report ID.
	reportID := checkInResp.WebConnectivity.ReportID
	logger.Infof("ReportID: %s", reportID)

	// Obtain experiment inputs.
	inputs := getInputs(args, checkInResp)

	// Create an instance of the experiment's measurer.
	measurer := &Measurer{Config: &Config{}}

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
		err := measureSingleURL(
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

// measureSingleURL measures a single URL.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation/timeout;
//
// - measurer is the measurer;
//
// - testStartTime is when the nettest started;
//
// - inputIdx is the input URL's index;
//
// - input is the current input;
//
// - reportID is the reportID to use.
func measureSingleURL(
	ctx context.Context,
	args *model.ExperimentMainArgs,
	measurer *Measurer,
	testStartTime time.Time,
	reportID string,
	inputIdx int,
	input *model.OOAPIURLInfo,
) error {
	sess := args.Session

	// Make sure we track this URL into the database.
	urlIdx, err := args.Database.CreateOrUpdateURL(
		input.URL,
		input.CategoryCode,
		input.CountryCode,
	)
	if err != nil {
		return err
	}

	// Create a measurement object inside of the database.
	dbMeas, err := args.Database.CreateMeasurement(
		sql.NullString{
			String: reportID,
			Valid:  true,
		},
		"web_connectivity",
		args.MeasurementDir,
		inputIdx,
		args.ResultID,
		sql.NullInt64{
			Int64: urlIdx,
			Valid: true,
		},
	)
	if err != nil {
		return err
	}

	// Create the measurement for this URL.
	meas := model.NewMeasurement(
		sess,
		measurer,
		reportID,
		input.URL,
		testStartTime,
		args.Annotations,
	)

	// Perform the measurement proper.
	err = measurer.Run(ctx, &model.ExperimentArgs{
		Callbacks:   args.Callbacks,
		Measurement: meas,
		Session:     sess,
	})

	// In case of error the measurement failed because of some
	// fundamental issue, so we don't want to submit.
	if err != nil {
		failure := err.Error()
		return args.Database.Failed(dbMeas, failure)
	}

	// Extract the measurement summary and store it inside the database.
	summary, err := measurer.GetSummaryKeys(meas)
	if err != nil {
		return err
	}

	err = args.Database.AddTestKeys(dbMeas, summary)
	if err != nil {
		return err
	}

	// Attempt to submit the measurement.
	err = submitOrStoreLocally(ctx, args, sess, meas, reportID, input.URL, dbMeas)
	if err != nil {
		return err
	}

	// Mark the measurement as done
	return args.Database.Done(dbMeas)
}

// submitOrStoreLocally submits the measurement or stores it locally.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation/timeout;
//
// - args contains the experiment's main arguments;
//
// - sess is the measurement session;
//
// - reportID is the reportID;
//
// - input is the possibly-empty input;
//
// - dbMeas is the database's view of the measurement.
//
// This function will return error only in case of fatal errors such as
// not being able to write onto the local disk.
func submitOrStoreLocally(
	ctx context.Context,
	args *model.ExperimentMainArgs,
	sess model.ExperimentSession,
	meas *model.Measurement,
	reportID string,
	input string,
	dbMeas *model.DatabaseMeasurement,
) error {
	logger := sess.Logger()

	if !args.NoCollector {
		// Submit the measurement to the OONI backend.
		err := sess.SubmitMeasurementV2(ctx, meas)
		if err == nil {
			logger.Infof(
				"Measurement: https://explorer.ooni.org/measurement/%s?input=%s",
				reportID,
				input,
			)
			return args.Database.UploadSucceeded(dbMeas)
		}

		// Handle the case where we could not submit the measurement.
		failure := err.Error()
		if err := args.Database.UploadFailed(dbMeas, failure); err != nil {
			return err
		}

		// Fallthrough and attempt to save measurement on disk
	}

	// Serialize to JSON.
	data, err := json.Marshal(meas)
	if err != nil {
		return err
	}

	// Write the measurement and return result.
	return os.WriteFile(dbMeas.MeasurementFilePath.String, data, 0600)
}
