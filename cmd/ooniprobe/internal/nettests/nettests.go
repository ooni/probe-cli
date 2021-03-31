package nettests

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/database"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/pkg/errors"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
}

// NewController creates a nettest controller
func NewController(
	nt Nettest, probe *ooni.Probe, res *database.Result, sess *engine.Session) *Controller {
	return &Controller{
		Probe:   probe,
		nt:      nt,
		res:     res,
		Session: sess,
	}
}

// Controller is passed to the run method of every Nettest
// each nettest instance has one controller
type Controller struct {
	Probe       *ooni.Probe
	Session     *engine.Session
	res         *database.Result
	nt          Nettest
	ntCount     int
	ntIndex     int
	ntStartTime time.Time // used to calculate the eta
	msmts       map[int64]*database.Measurement
	inputIdxMap map[int64]int64 // Used to map mk idx to database id

	// InputFiles optionally contains the names of the input
	// files to read inputs from (only for nettests that take
	// inputs, of course)
	InputFiles []string

	// Inputs contains inputs to be tested. These are specified
	// using the command line using the --input flag.
	Inputs []string

	// RunType contains the run_type hint for the CheckIn API. If
	// not set, the underlying code defaults to "timed".
	RunType string

	// numInputs is the total number of inputs
	numInputs int

	// curInputIdx is the current input index
	curInputIdx int
}

// SetInputIdxMap is used to set the mapping of index into input. This mapping
// is used to reference, for example, a particular URL based on the index inside
// of the input list and the index of it in the database.
func (c *Controller) SetInputIdxMap(inputIdxMap map[int64]int64) error {
	c.inputIdxMap = inputIdxMap
	return nil
}

// SetNettestIndex is used to set the current nettest index and total nettest
// count to compute a different progress percentage.
func (c *Controller) SetNettestIndex(i, n int) {
	c.ntCount = n
	c.ntIndex = i
}

// Run runs the selected nettest using the related experiment
// with the specified inputs.
//
// This function will continue to run in most cases but will
// immediately halt if something's wrong with the file system.
func (c *Controller) Run(builder *engine.ExperimentBuilder, inputs []string) error {
	// This will configure the controller as handler for the callbacks
	// called by ooni/probe-engine/experiment.Experiment.
	builder.SetCallbacks(model.ExperimentCallbacks(c))
	c.numInputs = len(inputs)
	exp := builder.NewExperiment()
	defer func() {
		c.res.DataUsageDown += exp.KibiBytesReceived()
		c.res.DataUsageUp += exp.KibiBytesSent()
	}()

	c.msmts = make(map[int64]*database.Measurement)

	// These values are shared by every measurement
	var reportID sql.NullString
	resultID := c.res.ID

	log.Debug(color.RedString("status.queued"))
	log.Debug(color.RedString("status.started"))

	if c.Probe.Config().Sharing.UploadResults {
		if err := exp.OpenReport(); err != nil {
			log.Debugf(
				"%s: %s", color.RedString("failure.report_create"), err.Error(),
			)
		} else {
			log.Debugf(color.RedString("status.report_create"))
			reportID = sql.NullString{String: exp.ReportID(), Valid: true}
		}
	}

	maxRuntime := time.Duration(c.Probe.Config().Nettests.WebsitesMaxRuntime) * time.Second
	if c.RunType == "timed" && maxRuntime > 0 {
		log.Debug("disabling maxRuntime when running in the background")
		maxRuntime = 0
	}
	start := time.Now()
	c.ntStartTime = start
	for idx, input := range inputs {
		if c.Probe.IsTerminated() {
			log.Info("user requested us to terminate using Ctrl-C")
			break
		}
		if maxRuntime > 0 && time.Since(start) > maxRuntime {
			log.Info("exceeded maximum runtime")
			break
		}
		c.curInputIdx = idx // allow for precise progress
		idx64 := int64(idx)
		log.Debug(color.RedString("status.measurement_start"))
		var urlID sql.NullInt64
		if c.inputIdxMap != nil {
			urlID = sql.NullInt64{Int64: c.inputIdxMap[idx64], Valid: true}
		}

		msmt, err := database.CreateMeasurement(
			c.Probe.DB(), reportID, exp.Name(), c.res.MeasurementDir, idx, resultID, urlID,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create measurement")
		}
		c.msmts[idx64] = msmt

		if input != "" {
			c.OnProgress(0, fmt.Sprintf("processing input: %s", input))
		}
		measurement, err := exp.Measure(input)
		if err != nil {
			log.WithError(err).Debug(color.RedString("failure.measurement"))
			if err := c.msmts[idx64].Failed(c.Probe.DB(), err.Error()); err != nil {
				return errors.Wrap(err, "failed to mark measurement as failed")
			}
			// Even with a failed measurement, we want to continue. We want to
			// record and submit the information we have. Saving the information
			// is useful for local inspection. Submitting it is useful to us to
			// understand what went wrong (censorship? bug? anomaly?).
		}

		saveToDisk := true
		if c.Probe.Config().Sharing.UploadResults {
			// Implementation note: SubmitMeasurement will fail here if we did fail
			// to open the report but we still want to continue. There will be a
			// bit of a spew in the logs, perhaps, but stopping seems less efficient.
			if err := exp.SubmitAndUpdateMeasurement(measurement); err != nil {
				log.Debug(color.RedString("failure.measurement_submission"))
				if err := c.msmts[idx64].UploadFailed(c.Probe.DB(), err.Error()); err != nil {
					return errors.Wrap(err, "failed to mark upload as failed")
				}
			} else if err := c.msmts[idx64].UploadSucceeded(c.Probe.DB()); err != nil {
				return errors.Wrap(err, "failed to mark upload as succeeded")
			} else {
				// Everything went OK, don't save to disk
				saveToDisk = false
			}
		}
		// We only save the measurement to disk if we failed to upload the measurement
		if saveToDisk {
			if err := exp.SaveMeasurement(measurement, msmt.MeasurementFilePath.String); err != nil {
				return errors.Wrap(err, "failed to save measurement on disk")
			}
		}

		if err := c.msmts[idx64].Done(c.Probe.DB()); err != nil {
			return errors.Wrap(err, "failed to mark measurement as done")
		}

		// We're not sure whether it's enough to log the error or we should
		// instead also mark the measurement as failed. Strictly speaking this
		// is an inconsistency between the code that generate the measurement
		// and the code that process the measurement. We do have some data
		// but we're not gonna have a summary. To be reconsidered.
		tk, err := exp.GetSummaryKeys(measurement)
		if err != nil {
			log.WithError(err).Error("failed to obtain testKeys")
			continue
		}
		log.Debugf("Fetching: %d %v", idx, c.msmts[idx64])
		if err := database.AddTestKeys(c.Probe.DB(), c.msmts[idx64], tk); err != nil {
			return errors.Wrap(err, "failed to add test keys to summary")
		}
	}

	log.Debugf("status.end")
	return nil
}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	// TODO(bassosimone): should we adjust this algorithm when we have a
	// maximum runtime that we would like to honor?
	log.Debugf("OnProgress: %f - %s", perc, msg)
	var eta float64
	eta = -1.0
	if c.numInputs > 1 {
		// make the percentage relative to the current input over all inputs
		floor := (float64(c.curInputIdx) / float64(c.numInputs))
		step := 1.0 / float64(c.numInputs)
		perc = floor + perc*step
		if c.curInputIdx > 0 {
			eta = (time.Since(c.ntStartTime).Seconds() / float64(c.curInputIdx)) * float64(c.numInputs-c.curInputIdx)
		}
	}
	if c.ntCount > 0 {
		// make the percentage relative to the current nettest over all nettests
		perc = float64(c.ntIndex)/float64(c.ntCount) + perc/float64(c.ntCount)
	}
	key := fmt.Sprintf("%T", c.nt)
	output.Progress(key, perc, eta, msg)
}
