package nettests

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/pkg/errors"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
}

// NewController creates a nettest controller
func NewController(
	nt Nettest, probe *ooni.Probe, res *model.DatabaseResult, sess *engine.Session) *Controller {
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
	res         *model.DatabaseResult
	nt          Nettest
	ntCount     int
	ntIndex     int
	ntStartTime time.Time // used to calculate the eta
	msmts       map[int64]*model.DatabaseMeasurement
	inputIdxMap map[int64]int64 // Used to map mk idx to database id

	// InputFiles optionally contains the names of the input
	// files to read inputs from (only for nettests that take
	// inputs, of course)
	InputFiles []string

	// Inputs contains inputs to be tested. These are specified
	// using the command line using the --input flag.
	Inputs []string

	// RunType contains the run_type hint for the CheckIn API. If
	// not set, the underlying code defaults to model.RunTypeTimed.
	RunType model.RunType

	// numInputs is the total number of inputs
	numInputs int

	// curInputIdx is the current input index
	curInputIdx int

	// saveToDisk indicates if we want to save the measurement to disk
	saveToDisk bool
}

// BuildAndSetInputIdxMap takes in input a list of URLs in the format
// returned by the check-in API (i.e., model.URLInfo) and performs
// the following actions:
//
// 1. inserts each URL into the database;
//
// 2. builds a list of bare URLs to be tested;
//
// 3. registers a mapping between each URL and an index
// and stores it into the controller.
//
// Arguments:
//
// - sess is the database in which to register the URL;
//
// - testlist is the result from the check-in API (or possibly
// a manually constructed list when applicable, e.g., for dnscheck
// until we have an API for serving its input).
//
// Results:
//
// - on success, a list of strings containing URLs to test;
//
// - on failure, an error.
func (c *Controller) BuildAndSetInputIdxMap(testlist []model.OOAPIURLInfo) ([]string, error) {
	var urls []string
	urlIDMap := make(map[int64]int64)
	for idx, url := range testlist {
		log.Debugf("Going over URL %d", idx)
		urlID, err := c.Probe.DB().CreateOrUpdateURL(
			url.URL, url.CategoryCode, url.CountryCode,
		)
		if err != nil {
			log.Error("failed to add to the URL table")
			return nil, err
		}
		log.Debugf("Mapped URL %s to idx %d and urlID %d", url.URL, idx, urlID)
		urlIDMap[int64(idx)] = urlID
		urls = append(urls, url.URL)
	}
	c.inputIdxMap = urlIDMap
	return urls, nil
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
func (c *Controller) Run(builder model.ExperimentBuilder, inputs []string) error {
	db := c.Probe.DB()
	// This will configure the controller as handler for the callbacks
	// called by ooni/probe-engine/experiment.Experiment.
	builder.SetCallbacks(model.ExperimentCallbacks(c))
	c.numInputs = len(inputs)
	exp := builder.NewExperiment()
	defer func() {
		c.res.DataUsageDown += exp.KibiBytesReceived()
		c.res.DataUsageUp += exp.KibiBytesSent()
	}()

	c.msmts = make(map[int64]*model.DatabaseMeasurement)

	// These values are shared by every measurement
	var reportID sql.NullString
	resultID := c.res.ID

	log.Debug(color.RedString("status.queued"))
	log.Debug(color.RedString("status.started"))

	maxRuntime := time.Duration(c.Probe.Config().Nettests.WebsitesMaxRuntime) * time.Second
	if c.RunType == model.RunTypeTimed && maxRuntime > 0 {
		log.Debug("disabling maxRuntime when running in the background")
		maxRuntime = 0
	}
	_, isWebConnectivity := c.nt.(WebConnectivity)
	if !isWebConnectivity {
		log.Debug("disabling maxRuntime without Web Connectivity")
		maxRuntime = 0
	}
	if len(c.Inputs) > 0 || len(c.InputFiles) > 0 {
		log.Debug("disabling maxRuntime with user-provided input")
		maxRuntime = 0
	}
	start := time.Now()
	c.ntStartTime = start

	ctx := context.Background()
	submitter, err := c.newSubmitter(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to initialise submitter")
	}

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

		// TODO(DecFox): we currently pass a nil reportID which should be replaced by the reportID
		// we get from the submitter. However, since the reportID is generated on uploading the first
		// measurement, we do not have a valid reportID to pass here. Therefore, we want to populate the
		// submitter without having to upload a measurement.
		msmt, err := db.CreateMeasurement(
			reportID, exp.Name(), c.res.MeasurementDir, idx, resultID, urlID,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create measurement")
		}
		c.msmts[idx64] = msmt

		if input != "" {
			c.OnProgress(0, fmt.Sprintf("processing input: %s", input))
		}
		measurement, err := exp.MeasureWithContext(context.Background(), input)
		if err != nil {
			log.WithError(err).Debug(color.RedString("failure.measurement"))
			if err := db.Failed(c.msmts[idx64], err.Error()); err != nil {
				return errors.Wrap(err, "failed to mark measurement as failed")
			}
			// Since https://github.com/ooni/probe-cli/pull/527, the Measure
			// function returns EITHER a valid measurement OR an error. Before
			// that, instead, the measurement was valid EVEN in case of an
			// error, which is quite not the <value> OR <error> semantics that
			// is so typical and widespread in the Go ecosystem. So, we must
			// jump to the next iteration of the loop here rather than falling
			// through and attempting to do something with the measurement.
			continue
		}

		c.saveToDisk = true

		// upload measurement
		if err := submitter.Submit(ctx, measurement); err != nil {
			log.Debug(color.RedString("failure.measurement_submission"))
			if err := db.UploadFailed(c.msmts[idx64], err.Error()); err != nil {
				return errors.Wrap(err, "failed to mark upload as failed")
			}
		} else if err := db.UploadSucceeded(c.msmts[idx64]); err != nil {
			return errors.Wrap(err, "failed to mark upload as succeeded")
		} else {
			// Everything went OK, don't save to disk
			c.saveToDisk = false
		}

		// We only save the measurement to disk if we failed to upload the measurement
		saver, err := c.newSaver(exp, *msmt)
		if err != nil {
			return errors.Wrap(err, "failed to initialise saver")
		}
		err = saver.SaveMeasurement(measurement)
		if err != nil {
			return errors.Wrap(err, "failed to save measurement on disk")
		}

		if err := db.Done(c.msmts[idx64]); err != nil {
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
		if err := db.AddTestKeys(c.msmts[idx64], tk); err != nil {
			return errors.Wrap(err, "failed to add test keys to summary")
		}
	}
	db.UpdateUploadedStatus(c.res)
	log.Debugf("status.end")
	return nil
}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	// when we have maxRuntime, honor it
	maxRuntime := time.Duration(c.Probe.Config().Nettests.WebsitesMaxRuntime) * time.Second
	_, isWebConnectivity := c.nt.(WebConnectivity)
	userProvidedInput := len(c.Inputs) > 0 || len(c.InputFiles) > 0
	if c.RunType == model.RunTypeManual && maxRuntime > 0 && isWebConnectivity && !userProvidedInput {
		elapsed := time.Since(c.ntStartTime)
		perc = float64(elapsed) / float64(maxRuntime)
		eta := maxRuntime.Seconds() - elapsed.Seconds()
		log.Debugf("OnProgress: %f - %s", perc, msg)
		key := fmt.Sprintf("%T", c.nt)
		output.Progress(key, perc, eta, msg)
		return
	}
	// otherwise estimate the ETA
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

// newSaver creates a new engine.Saver instance.
func (c *Controller) newSaver(experiment model.Experiment, msmt model.DatabaseMeasurement) (engine.Saver, error) {
	return engine.NewSaver(engine.SaverConfig{
		Enabled:    c.saveToDisk,
		Experiment: experiment,
		FilePath:   msmt.MeasurementFilePath.String,
		Logger:     c.Session.Logger(),
	})
}

// newSubmitter creates a new engine.Submitter instance.
func (c *Controller) newSubmitter(ctx context.Context) (engine.Submitter, error) {
	return engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: c.Probe.Config().Sharing.UploadResults,
		Session: c.Session,
		Logger:  c.Session.Logger(),
	})
}
