package riseupvpn

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
)

// ErrNoCheckInInfo indicates check-in returned no suitable info.
var ErrNoCheckInInfo = errors.New("riseupvpn: returned no check-in info")

// ExperimentMain
type ExperimentMain struct {
	options    model.ExperimentOptions
	args       *model.ExperimentMainArgs
	configArgs *Config
}

// SetOptions
func (em *ExperimentMain) SetOptions(options model.ExperimentOptions) {
	em.options = options
}

// SetArguments
func (em *ExperimentMain) SetArguments(sess model.ExperimentSession, db *model.DatabaseProps, extraOptions map[string]any) error {
	em.args = em.options.SetArguments(sess, db)
	em.configArgs = &Config{}
	// give precedence to options populated by OONIRun
	if extraOptions == nil {
		extraOptions = em.options.ExtraOptions()
	}
	err := setter.SetOptionsAny(em.configArgs, extraOptions)
	return err
}

// Main is the main function of the experiment.
func (em *ExperimentMain) Main(ctx context.Context) error {
	sess := em.args.Session
	logger := sess.Logger()

	// Create the directory where to store results unless it already exists
	if err := os.MkdirAll(em.args.MeasurementDir, 0700); err != nil {
		return err
	}

	// Attempt to remove the results directory when done unless it
	// contains files, in which case we should keep it.
	defer os.Remove(em.args.MeasurementDir)

	// Call the check-in API to obtain configuration. Note that the value
	// returned here MAY have been cached by the engine.
	logger.Infof("calling check-in API...")
	checkInResp, err := experiment.CallCheckIn(ctx, em.args, sess)

	// Bail if either the check-in API failed or we don't have a reportID
	// with which to submit Web Connectivity measurements results.
	if err != nil {
		return err
	}
	if checkInResp.RiseupVPN == nil {
		return ErrNoCheckInInfo
	}

	// Obtain and log the report ID.
	reportID := checkInResp.RiseupVPN.ReportID
	logger.Infof("ReportID: %s", reportID)

	// Create an instance of the experiment's measurer.
	measurer := &Measurer{Config: *em.configArgs}

	// Record when we started running this nettest.
	testStartTime := time.Now()

	return experiment.MeasurePossiblyNilInput(
		ctx,
		em.args,
		measurer,
		testStartTime,
		reportID,
		0,   // inputIdx
		nil, // input
	)
}
