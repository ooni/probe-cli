package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrNoCheckInInfo indicates check-in returned no suitable info.
var ErrNoCheckInInfo = errors.New("webconnectivity: returned no check-in info")

// TODO(bassosimone): I am wondering whether we should have a specific
// MainArgs struct for each experiment rather than a common struct.

// TODO(bassosimone): ideally, I would like OONI Run v2 to call Main.

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
	if checkInResp.Telegram == nil {
		return ErrNoCheckInInfo
	}

	// Obtain and log the report ID.
	reportID := checkInResp.Telegram.ReportID
	logger.Infof("ReportID: %s", reportID)

	// Create an instance of the experiment's measurer.
	measurer := &Measurer{Config: Config{}}

	// Record when we started running this nettest.
	testStartTime := time.Now()

	// Create a measurement object inside of the database.
	dbMeas, err := args.Database.CreateMeasurement(
		sql.NullString{
			String: reportID,
			Valid:  true,
		},
		"telegram",
		args.MeasurementDir,
		0,
		args.ResultID,
		sql.NullInt64{
			Int64: 0,
			Valid: false,
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
		"",
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
	err = submitOrStoreLocally(ctx, args, sess, meas, reportID, "", dbMeas)
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

	// TODO(bassosimone): this function is basically the same for each
	// experiment so we can easily share it. The only "tricky" part
	// here is that we should construct the explorer URL differently
	// depending on whether there's input.

	if !args.NoCollector {
		// Submit the measurement to the OONI backend.
		err := sess.SubmitMeasurementV2(ctx, meas)
		if err == nil {
			logger.Infof(
				"Measurement: https://explorer.ooni.org/measurement/%s",
				reportID,
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
