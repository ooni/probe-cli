// Package experiment contains common code for implementing experiments.
package experiment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CallCheckIn is a convenience function that calls the
// check-in API using the given arguments.
func CallCheckIn(
	ctx context.Context,
	args *model.ExperimentMainArgs,
	sess model.ExperimentSession,
) (*model.OOAPICheckInNettests, error) {
	return sess.CheckIn(ctx, &model.OOAPICheckInConfig{
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
}

// MeasurePossiblyNilInput measures a possibly nil
// input using the given experiment measurer.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation/timeout;
//
// - args contains the experiment-main's arguments;
//
// - measurer is the measurer;
//
// - testStartTime is when the nettest started;
//
// - reportID is the reportID to use;
//
// - inputIdx is the POSSIBLY-ZERO input's index;
//
// - input is the POSSIBLY-NIL input to measure.
//
// This function only returns an error in case there is a
// serious situation (e.g., cannot write to disk).
func MeasurePossiblyNilInput(
	ctx context.Context,
	args *model.ExperimentMainArgs,
	measurer model.ExperimentMeasurer,
	testStartTime time.Time,
	reportID string,
	inputIdx int,
	input *model.OOAPIURLInfo,
) error {
	runtimex.Assert(ctx != nil, "passed nil Context")
	runtimex.Assert(args != nil, "passed nil ExperimentMainArgs")
	runtimex.Assert(measurer != nil, "passed nil ExperimentMeasurer")
	runtimex.Assert(reportID != "", "passed empty report ID")
	sess := args.Session

	// Make sure we track this possibly-nil input into the database.
	var (
		urlIdx   sql.NullInt64
		urlInput string
	)
	if input != nil {
		index, err := args.Database.CreateOrUpdateURL(
			input.URL,
			input.CategoryCode,
			input.CountryCode,
		)
		if err != nil {
			return err
		}
		urlIdx.Int64 = index
		urlIdx.Valid = true
		urlInput = input.URL
	}

	// Create a measurement object inside of the database.
	dbMeas, err := args.Database.CreateMeasurement(
		sql.NullString{
			String: reportID,
			Valid:  true,
		},
		measurer.ExperimentName(),
		args.MeasurementDir,
		inputIdx,
		args.ResultID,
		urlIdx,
	)
	if err != nil {
		return err
	}

	// Create the measurement for this URL.
	meas := model.NewMeasurement(
		sess,
		measurer,
		reportID,
		urlInput, // possibly the empty string
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
		return args.Database.Failed(dbMeas, err.Error())
	}

	// Extract the measurement summary and store it inside the database.
	summary, err := measurer.GetSummaryKeys(meas)
	if err != nil {
		return err
	}

	// Add summary to database.
	err = args.Database.AddTestKeys(dbMeas, summary)
	if err != nil {
		return err
	}

	// Attempt to submit the measurement.
	err = SubmitOrStoreLocally(ctx, args, sess, meas, dbMeas)
	if err != nil {
		return err
	}

	// Mark measurement as done.
	return args.Database.Done(dbMeas)
}

// SubmitOrStoreLocally submits the measurement or stores it locally.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation/timeout;
//
// - args contains the experiment's main arguments;
//
// - sess is the measurement session;
//
// - dbMeas is the database's view of the measurement.
//
// This function will return error only in case of fatal errors such as
// not being able to write onto the local disk.
func SubmitOrStoreLocally(
	ctx context.Context,
	args *model.ExperimentMainArgs,
	sess model.ExperimentSession,
	meas *model.Measurement,
	dbMeas *model.DatabaseMeasurement,
) error {
	runtimex.Assert(args != nil, "passed nil arguments")
	runtimex.Assert(sess != nil, "passed nil Session")
	runtimex.Assert(meas != nil, "passed nil measurement")
	runtimex.Assert(dbMeas != nil, "passed nil dbMeas")
	logger := sess.Logger()

	if !args.NoCollector {
		// Submit the measurement to the OONI backend.
		err := sess.SubmitMeasurementV2(ctx, meas)
		if err == nil {
			logger.Infof("Measurement: %s", ExplorerURL(meas))
			return args.Database.UploadSucceeded(dbMeas)
		}

		// Handle the case where we could not submit the measurement.
		failure := err.Error()
		if err := args.Database.UploadFailed(dbMeas, failure); err != nil {
			return err
		}

		// Fallthrough and attempt to save measurement to disk
	}

	// Serialize to JSON.
	data, err := json.Marshal(meas)
	if err != nil {
		return err
	}

	// Write the measurement and return result.
	return os.WriteFile(dbMeas.MeasurementFilePath.String, data, 0600)
}

// ExplorerURL returns the explorer URL associated with a measurement.
func ExplorerURL(meas *model.Measurement) string {
	runtimex.Assert(meas != nil, "passed nil measurement")
	runtimex.Assert(meas.ReportID != "", "passed empty report ID")
	URL := &url.URL{
		Scheme: "https",
		Host:   "explorer.ooni.org",
		Path:   fmt.Sprintf("/measurement/%s", meas.ReportID),
	}
	if meas.Input != "" {
		query := url.Values{}
		query.Add("input", string(meas.Input))
		URL.RawQuery = query.Encode()
	}
	return URL.String()
}
