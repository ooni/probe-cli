package nettests

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/output"
	"github.com/ooni/probe-cli/utils"
	engine "github.com/ooni/probe-engine"
	"github.com/pkg/errors"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	GetTestKeys(map[string]interface{}) (interface{}, error)
	LogSummary(string) error
}

// NewController creates a nettest controller
func NewController(nt Nettest, ctx *ooni.Context, res *database.Result) *Controller {
	msmtPath := filepath.Join(ctx.TempDir,
		fmt.Sprintf("msmt-%T-%s.jsonl", nt,
			time.Now().UTC().Format(utils.ResultTimestamp)))
	return &Controller{
		Ctx:      ctx,
		nt:       nt,
		res:      res,
		msmtPath: msmtPath,
	}
}

// Controller is passed to the run method of every Nettest
// each nettest instance has one controller
type Controller struct {
	Ctx         *ooni.Context
	res         *database.Result
	nt          Nettest
	ntCount     int
	ntIndex     int
	ntStartTime time.Time // used to calculate the eta
	msmts       map[int64]*database.Measurement
	msmtPath    string          // XXX maybe we can drop this and just use a temporary file
	inputIdxMap map[int64]int64 // Used to map mk idx to database id

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
	builder.SetCallbacks(engine.Callbacks(c))
	c.numInputs = len(inputs)
	exp := builder.Build()

	c.msmts = make(map[int64]*database.Measurement)

	// These values are shared by every measurement
	var reportID sql.NullString
	resultID := c.res.ID

	log.Debug(color.RedString("status.queued"))
	log.Debug(color.RedString("status.started"))
	log.Debugf("OutputPath: %s", c.msmtPath)

	if c.Ctx.Config.Sharing.UploadResults {
		if err := exp.OpenReport(); err != nil {
			log.Debugf(
				"%s: %s", color.RedString("failure.report_create"), err.Error(),
			)
		} else {
			defer exp.CloseReport()
			log.Debugf(color.RedString("status.report_create"))
			reportID = sql.NullString{String: exp.ReportID(), Valid: true}
		}
	}

	c.ntStartTime = time.Now()
	for idx, input := range inputs {
		c.curInputIdx = idx // allow for precise progress
		idx64 := int64(idx)
		log.Debug(color.RedString("status.measurement_start"))
		var urlID sql.NullInt64
		if c.inputIdxMap != nil {
			urlID = sql.NullInt64{Int64: c.inputIdxMap[idx64], Valid: true}
		}
		msmt, err := database.CreateMeasurement(
			c.Ctx.DB, reportID, exp.Name(), resultID, c.msmtPath, urlID,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create measurement")
		}
		c.msmts[idx64] = msmt

		measurement, err := exp.Measure(input)
		if err != nil {
			log.WithError(err).Debug(color.RedString("failure.measurement"))
			if err := c.msmts[idx64].Failed(c.Ctx.DB, err.Error()); err != nil {
				return errors.Wrap(err, "failed to mark measurement as failed")
			}
			continue
		}

		if c.Ctx.Config.Sharing.UploadResults {
			// Implementation note: SubmitMeasurement will fail here if we did fail
			// to open the report but we still want to continue. There will be a
			// bit of a spew in the logs, perhaps, but stopping seems less efficient.
			if err := exp.SubmitAndUpdateMeasurement(measurement); err != nil {
				log.Debug(color.RedString("failure.measurement_submission"))
				if err := c.msmts[idx64].UploadFailed(c.Ctx.DB, err.Error()); err != nil {
					return errors.Wrap(err, "failed to mark upload as failed")
				}
			} else if err := c.msmts[idx64].UploadSucceeded(c.Ctx.DB); err != nil {
				return errors.Wrap(err, "failed to mark upload as succeeded")
			}
		}

		if err := exp.SaveMeasurement(measurement, c.msmtPath); err != nil {
			return errors.Wrap(err, "failed to save measurement on disk")
		}
		if err := c.msmts[idx64].Done(c.Ctx.DB); err != nil {
			return errors.Wrap(err, "failed to mark measurement as done")
		}

		// We're not sure whether it's enough to log the error or we should
		// instead also mark the measurement as failed. Strictly speaking this
		// is an inconsistency between the code that generate the measurement
		// and the code that process the measurement. We do have some data
		// but we're not gonna have a summary. To be reconsidered.
		genericTk, err := measurement.MakeGenericTestKeys()
		if err != nil {
			log.WithError(err).Error("failed to cast the test keys")
			continue
		}
		tk, err := c.nt.GetTestKeys(genericTk)
		if err != nil {
			log.WithError(err).Error("failed to obtain testKeys")
			continue
		}
		log.Debugf("Fetching: %d %v", idx, c.msmts[idx64])
		if err := database.AddTestKeys(c.Ctx.DB, c.msmts[idx64], tk); err != nil {
			return errors.Wrap(err, "failed to add test keys to summary")
		}
	}

	log.Debugf("status.end")
	for idx, msmt := range c.msmts {
		log.Debugf("adding msmt#%d to result", idx)
		if err := msmt.AddToResult(c.Ctx.DB, c.res); err != nil {
			return errors.Wrap(err, "failed to add to result")
		}
	}
	return nil
}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	log.Debugf("OnProgress: %f - %s", perc, msg)
	var eta float64
	eta = -1.0
	if c.numInputs >= 1 {
		// make the percentage relative to the current input over all inputs
		floor := (float64(c.curInputIdx) / float64(c.numInputs))
		step := 1.0 / float64(c.numInputs)
		perc = floor + perc*step
		eta = (float64(time.Now().Sub(c.ntStartTime).Seconds()) / float64(c.curInputIdx)) * float64(c.numInputs-c.curInputIdx)
	}
	if c.ntCount > 0 {
		// make the percentage relative to the current nettest over all nettests
		perc = float64(c.ntIndex)/float64(c.ntCount) + perc/float64(c.ntCount)
	}
	key := fmt.Sprintf("%T", c.nt)
	output.Progress(key, perc, eta, msg)
}

// OnDataUsage should be called when we have a data usage update.
func (c *Controller) OnDataUsage(dloadKiB, uploadKiB float64) {
	c.res.DataUsageDown += dloadKiB
	c.res.DataUsageUp += uploadKiB
}
