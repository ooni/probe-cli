package nettests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/measurement-kit/go-measurement-kit"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/crashreport"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/output"
	"github.com/ooni/probe-cli/utils"
	"github.com/ooni/probe-cli/utils/strcase"
	"github.com/ooni/probe-cli/version"
	"github.com/ooni/probe-engine/experiment"
	"github.com/ooni/probe-engine/experiment/handler"
	"github.com/pkg/errors"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	GetTestKeys(interface{}) (interface{}, error)
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
	msmts       map[int64]*database.Measurement
	msmtPath    string          // XXX maybe we can drop this and just use a temporary file
	inputIdxMap map[int64]int64 // Used to map mk idx to database id
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

// Init should be called once to initialise the nettest
//
// This is the codepath for running MK nettests.
func (c *Controller) Init(nt *mk.Nettest) error {
	log.Debugf("Init: %v", nt)
	err := c.Ctx.MaybeLocationLookup()
	if err != nil {
		return err
	}

	c.msmts = make(map[int64]*database.Measurement)

	// These values are shared by every measurement
	reportID := sql.NullString{String: "", Valid: false}
	testName := strcase.ToSnake(nt.Name)
	resultID := c.res.ID
	reportFilePath := c.msmtPath
	geoIPCountryPath := c.Ctx.Session.CountryDatabasePath()
	geoIPASNPath := c.Ctx.Session.ASNDatabasePath()
	msmtPath := c.msmtPath

	log.Debugf("OutputPath: %s", msmtPath)
	nt.Options = mk.NettestOptions{
		IncludeIP:      c.Ctx.Config.Sharing.IncludeIP,
		IncludeASN:     c.Ctx.Config.Sharing.IncludeASN,
		IncludeCountry: c.Ctx.Config.Sharing.IncludeCountry,
		LogLevel:       "INFO",

		ProbeCC:  c.Ctx.Session.ProbeCC(),
		ProbeASN: c.Ctx.Session.ProbeASNString(),
		ProbeIP:  c.Ctx.Session.ProbeIP(),

		DisableReportFile: false,
		DisableCollector:  !c.Ctx.Config.Sharing.UploadResults,
		RandomizeInput:    false, // It's important to disable input randomization to ensure the URLs are written in sync to the DB
		SoftwareName:      "ooniprobe-desktop",
		SoftwareVersion:   version.Version,
		CollectorBaseURL:  c.Ctx.Config.Advanced.CollectorURL,
		BouncerBaseURL:    c.Ctx.Config.Advanced.BouncerURL,

		OutputPath:       msmtPath,
		GeoIPCountryPath: geoIPCountryPath,
		GeoIPASNPath:     geoIPASNPath,
		CaBundlePath:     c.Ctx.Session.CABundlePath(),
	}

	log.Debugf("CaBundlePath: %s", nt.Options.CaBundlePath)
	log.Debugf("GeoIPASNPath: %s", nt.Options.GeoIPASNPath)
	log.Debugf("GeoIPCountryPath: %s", nt.Options.GeoIPCountryPath)

	nt.On("log", func(e mk.Event) {
		level := e.Value.LogLevel
		msg := e.Value.Message

		switch level {
		case "ERROR":
			log.Errorf("%v: %s", color.RedString("mklog"), msg)
		case "INFO":
			log.Infof("%v: %s", color.BlueString("mklog"), msg)
		default:
			log.Debugf("%v: %s", color.WhiteString("mklog"), msg)
		}

	})

	nt.On("status.queued", func(e mk.Event) {
		log.Debugf("%s", e.Key)
	})

	nt.On("status.started", func(e mk.Event) {
		log.Debugf("%s", e.Key)
	})

	nt.On("status.report_create", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		reportID = sql.NullString{String: e.Value.ReportID, Valid: true}
	})

	nt.On("failure.report_create", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})

	nt.On("status.geoip_lookup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
	})

	nt.On("status.measurement_start", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		idx := e.Value.Idx
		urlID := sql.NullInt64{Int64: 0, Valid: false}
		if c.inputIdxMap != nil {
			urlID = sql.NullInt64{Int64: c.inputIdxMap[idx], Valid: true}
		}
		msmt, err := database.CreateMeasurement(c.Ctx.DB, reportID, testName, resultID, reportFilePath, urlID)
		if err != nil {
			log.WithError(err).Error("Failed to create measurement")
			return
		}
		c.msmts[idx] = msmt
	})

	nt.On("status.progress", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		perc := e.Value.Percentage
		if c.ntCount > 0 {
			perc = float64(c.ntIndex)/float64(c.ntCount) + perc/float64(c.ntCount)
		}
		c.OnProgress(perc, e.Value.Message)
	})

	nt.On("status.update.*", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
	})

	// XXX should these be made into permanent failures?
	nt.On("failure.asn_lookup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})
	nt.On("failure.cc_lookup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})
	nt.On("failure.ip_lookup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})

	nt.On("failure.resolver_lookup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})

	nt.On("failure.report_close", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))
		log.Debugf("%v", e.Value)
	})

	nt.On("failure.startup", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		c.msmts[e.Value.Idx].Failed(c.Ctx.DB, e.Value.Failure)
	})

	nt.On("failure.measurement", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		c.msmts[e.Value.Idx].Failed(c.Ctx.DB, e.Value.Failure)
	})

	nt.On("failure.measurement_submission", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		failure := e.Value.Failure
		c.msmts[e.Value.Idx].UploadFailed(c.Ctx.DB, failure)
	})

	nt.On("status.measurement_submission", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		// XXX maybe this should change once MK is aligned with the spec
		if c.Ctx.Config.Sharing.UploadResults == true {
			if err := c.msmts[e.Value.Idx].UploadSucceeded(c.Ctx.DB); err != nil {
				log.WithError(err).Error("failed to mark msmt as uploaded")
			}
		}
	})

	nt.On("status.measurement_done", func(e mk.Event) {
		log.Debugf(color.RedString(e.Key))

		if err := c.msmts[e.Value.Idx].Done(c.Ctx.DB); err != nil {
			log.WithError(err).Error("failed to mark msmt as done")
		}
	})

	nt.On("measurement", func(e mk.Event) {
		log.Debugf("status.end")

		crashreport.CapturePanicAndWait(func() {
			c.OnEntry(e.Value.Idx, e.Value.JSONStr)
		}, nil)
	})

	nt.On("status.end", func(e mk.Event) {
		log.Debugf("status.end")

		for idx, msmt := range c.msmts {
			log.Debugf("adding msmt#%d to result", idx)
			if err := msmt.AddToResult(c.Ctx.DB, c.res); err != nil {
				log.WithError(err).Error("failed to add to result")
			}
		}

		if e.Value.Failure != "" {
			log.Errorf("Failure in status.end: %s", e.Value.Failure)
		}

		c.res.DataUsageDown += e.Value.DownloadedKB
		c.res.DataUsageUp += e.Value.UploadedKB
	})

	log.Debugf("Registered all the handlers")
	return nil
}

// Run runs the selected nettest using the related experiment
// with the specified inputs.
//
// This function will continue to run in most cases but will
// immediately halt if something's wrong with the DB.
//
// This is the codepath for running ooni/probe-engine nettests.
func (c *Controller) Run(exp *experiment.Experiment, inputs []string) error {
	ctx := context.Background()

	// This will configure the controller as handler for the callbacks
	// called by ooni/probe-engine/experiment.Experiment.
	exp.Callbacks = handler.Callbacks(c)

	c.msmts = make(map[int64]*database.Measurement)

	// These values are shared by every measurement
	reportID := sql.NullString{String: "", Valid: false}
	resultID := c.res.ID

	log.Debug(color.RedString("status.queued"))
	log.Debug(color.RedString("status.started"))
	log.Debugf("OutputPath: %s", c.msmtPath)

	// XXX: double check that we're passing the right options to MK?

	if c.Ctx.Config.Sharing.UploadResults {
		if err := exp.OpenReport(ctx); err != nil {
			log.Debugf(
				"%s: %s", color.RedString("failure.report_create"), err.Error(),
			)
		} else {
			defer exp.CloseReport(ctx)
			log.Debugf(color.RedString("status.report_create"))
			reportID = sql.NullString{String: exp.ReportID(), Valid: true}
		}
	}

	for idx, input := range inputs {
		idx64 := int64(idx)
		log.Debug(color.RedString("status.measurement_start"))
		urlID := sql.NullInt64{Int64: 0, Valid: false}
		if c.inputIdxMap != nil {
			urlID = sql.NullInt64{Int64: c.inputIdxMap[idx64], Valid: true}
		}
		msmt, err := database.CreateMeasurement(
			c.Ctx.DB, reportID, exp.TestName, resultID, c.msmtPath, urlID,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create measurement")
		}
		c.msmts[idx64] = msmt

		measurement, err := exp.Measure(ctx, input)
		if err != nil {
			log.Debug(color.RedString("failure.measurement"))
			if err := c.msmts[idx64].Failed(c.Ctx.DB, err.Error()); err != nil {
				return errors.Wrap(err, "failed to mark measurement as failed")
			}
			continue
		}

		if c.Ctx.Config.Sharing.UploadResults {
			// Implementation note: SubmitMeasurement will fail here if we did fail
			// to open the report but we still want to continue. There will be a
			// bit of a spew in the logs, perhaps, but stopping seems less efficient.
			if err := exp.SubmitMeasurement(ctx, &measurement); err != nil {
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
		tk, err := c.nt.GetTestKeys(measurement.TestKeys)
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

	key := fmt.Sprintf("%T", c.nt)
	output.Progress(key, perc, msg)
}

// OnDataUsage should be called when we have a data usage update.
func (c *Controller) OnDataUsage(dloadKiB, uploadKiB float64) {
	c.res.DataUsageDown += dloadKiB
	c.res.DataUsageUp += uploadKiB
}

// Entry is an opaque measurement entry
type Entry struct {
	TestKeys map[string]interface{} `json:"test_keys"`
}

// OnEntry should be called every time there is a new entry
func (c *Controller) OnEntry(idx int64, jsonStr string) {
	log.Debugf("OnEntry")

	var entry Entry
	if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
		log.WithError(err).Error("failed to parse onEntry")
		return
	}
	// XXX is it correct to just log the error instead of marking the whole
	// measurement as failed?
	tk, err := c.nt.GetTestKeys(entry.TestKeys)
	if err != nil {
		log.WithError(err).Error("failed to obtain testKeys")
	}

	log.Debugf("Fetching: %d %v", idx, c.msmts[idx])
	err = database.AddTestKeys(c.Ctx.DB, c.msmts[idx], tk)
	if err != nil {
		log.WithError(err).Error("failed to add test keys to summary")
	}
}

// MKStart is the interface for the mk.Nettest Start() function
type MKStart func(name string) (chan bool, error)

// Start should be called to start the test
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
