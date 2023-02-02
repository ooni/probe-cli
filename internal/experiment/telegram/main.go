package telegram

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrNoCheckInInfo indicates check-in returned no suitable info.
var ErrNoCheckInInfo = errors.New("telegram: returned no check-in info")

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
	if checkInResp.Telegram == nil {
		return ErrNoCheckInInfo
	}

	// Obtain and log the report ID.
	reportID := checkInResp.Telegram.ReportID
	logger.Infof("ReportID: %s", reportID)

	// Create an instance of the experiment's measurer.
	measurer := &Measurer{Config: *config}

	// Record when we started running this nettest.
	testStartTime := time.Now()

	return experiment.MeasurePossiblyNilInput(
		ctx,
		args,
		measurer,
		testStartTime,
		reportID,
		0,   // inputIdx
		nil, // input
	)
}
