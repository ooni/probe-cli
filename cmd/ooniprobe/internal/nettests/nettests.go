package nettests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
	"github.com/ooni/probe-cli/v3/internal/miniengine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/pkg/errors"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
}

// NewController creates a nettest controller
func NewController(
	nt Nettest, probe *ooni.Probe, res *model.DatabaseResult, sess *miniengine.Session) *Controller {
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
	Session     *miniengine.Session
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

var _ model.ExperimentCallbacks = &Controller{}

// Run runs the selected nettest using the related experiment
// with the specified inputs.
//
// This function will continue to run in most cases but will
// immediately halt if something's wrong with the file system.
func (c *Controller) Run(
	experimentName string,
	checkInReportID string,
	inputs []string,
) error {
	db := c.Probe.DB()

	c.msmts = make(map[int64]*model.DatabaseMeasurement)

	// These values are shared by every measurement
	var reportID sql.NullString
	resultID := c.res.ID

	log.Debug(color.RedString("status.queued"))
	log.Debug(color.RedString("status.started"))

	canSubmit := c.Probe.Config().Sharing.UploadResults && checkInReportID != ""
	if canSubmit {
		reportID = sql.NullString{String: checkInReportID, Valid: true}
	}

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

		msmt, err := db.CreateMeasurement(
			reportID, experimentName, c.res.MeasurementDir, idx, resultID, urlID,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create measurement")
		}
		c.msmts[idx64] = msmt

		if input != "" {
			c.OnProgress(0, fmt.Sprintf("processing input: %s", input))
		}
		options := make(map[string]any)
		measurementTask := c.Session.Measure(context.Background(), experimentName, options, input)
		awaitTask(measurementTask, c)
		measurementResult, err := measurementTask.Result()
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

		// update the data usage counters
		c.res.DataUsageDown += measurementResult.KibiBytesReceived
		c.res.DataUsageUp += measurementResult.KibiBytesSent

		// set the measurement's reportID
		measurementResult.Measurement.ReportID = checkInReportID

		saveToDisk := true
		if canSubmit {
			// Implementation note: SubmitMeasurement will fail here if we did fail
			// to open the report but we still want to continue. There will be a
			// bit of a spew in the logs, perhaps, but stopping seems less efficient.
			submitTask := c.Session.Submit(context.Background(), measurementResult.Measurement)
			awaitTask(submitTask, model.NewPrinterCallbacks(taskLogger))
			if _, err := submitTask.Result(); err != nil {
				log.Debug(color.RedString("failure.measurement_submission"))
				if err := db.UploadFailed(c.msmts[idx64], err.Error()); err != nil {
					return errors.Wrap(err, "failed to mark upload as failed")
				}
			} else if err := db.UploadSucceeded(c.msmts[idx64]); err != nil {
				return errors.Wrap(err, "failed to mark upload as succeeded")
			} else {
				// Everything went OK, don't save to disk
				saveToDisk = false
			}
		}
		// We only save the measurement to disk if we failed to upload the measurement
		if saveToDisk {
			if err := c.saveMeasurement(measurementResult.Measurement, msmt.MeasurementFilePath.String); err != nil {
				return errors.Wrap(err, "failed to save measurement on disk")
			}
		}

		// make the measurement as done
		if err := db.Done(c.msmts[idx64]); err != nil {
			return errors.Wrap(err, "failed to mark measurement as done")
		}

		// write the measurement summary into the database
		if err := db.AddTestKeys(c.msmts[idx64], measurementResult.Summary); err != nil {
			return errors.Wrap(err, "failed to add test keys to summary")
		}
	}

	db.UpdateUploadedStatus(c.res)
	log.Debugf("status.end")
	return nil
}

// saveMeasurement saves a measurement to disk
func (c *Controller) saveMeasurement(meas *model.Measurement, filepath string) error {
	data, err := json.Marshal(meas)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0600)
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
