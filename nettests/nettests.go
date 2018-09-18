package nettests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	GetTestKeys(map[string]interface{}) interface{}
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
	msmts       map[int64]*database.Measurement
	msmtPath    string          // XXX maybe we can drop this and just use a temporary file
	inputIdxMap map[int64]int64 // Used to map mk idx to database id
}

func getCaBundlePath() string {
	path := os.Getenv("SSL_CERT_FILE")
	if path != "" {
		return path
	}
	return "/etc/ssl/cert.pem"
}

func (c *Controller) SetInputIdxMap(inputIdxMap map[int64]int64) error {
	c.inputIdxMap = inputIdxMap
	return nil
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) error {
	log.Debugf("Init: %v", nt)
	c.Ctx.LocationLookup()

	c.msmts = make(map[int64]*database.Measurement)

	// These values are shared by every measurement
	reportID := sql.NullString{String: "", Valid: false}
	testName := strcase.ToSnake(nt.Name)
	resultID := c.res.ID
	reportFilePath := c.msmtPath
	geoIPCountryPath := filepath.Join(utils.GeoIPDir(c.Ctx.Home), "GeoIP.dat")
	geoIPASNPath := filepath.Join(utils.GeoIPDir(c.Ctx.Home), "GeoIPASNum.dat")
	caBundlePath := getCaBundlePath()
	msmtPath := c.msmtPath

	log.Debugf("OutputPath: %s", msmtPath)
	nt.Options = mk.NettestOptions{
		IncludeIP:      c.Ctx.Config.Sharing.IncludeIP,
		IncludeASN:     c.Ctx.Config.Sharing.IncludeASN,
		IncludeCountry: c.Ctx.Config.Sharing.IncludeCountry,
		LogLevel:       "DEBUG",

		ProbeCC:  c.Ctx.Location.CountryCode,
		ProbeASN: fmt.Sprintf("AS%d", c.Ctx.Location.ASN),
		ProbeIP:  c.Ctx.Location.IP,

		DisableReportFile: false,
		DisableCollector:  !c.Ctx.Config.Sharing.UploadResults,
		RandomizeInput:    false, // It's important to disable input randomization to ensure the URLs are written in sync to the DB
		SoftwareName:      "ooniprobe-desktop",
		SoftwareVersion:   ooni.Version,

		OutputPath:       msmtPath,
		GeoIPCountryPath: geoIPCountryPath,
		GeoIPASNPath:     geoIPASNPath,
		CaBundlePath:     caBundlePath,
	}
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

	nt.On("status.report_created", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		reportID = sql.NullString{String: e.Value.ReportID, Valid: true}
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
		c.OnProgress(e.Value.Percentage, e.Value.Message)
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

	nt.On("failure.report_create", func(e mk.Event) {
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

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	log.Debugf("OnProgress: %f - %s", perc, msg)

	key := fmt.Sprintf("%T", c.nt)
	output.Progress(key, perc, msg)
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
	tk := c.nt.GetTestKeys(entry.TestKeys)

	log.Debugf("Fetching: %s %v", idx, c.msmts[idx])
	err := database.AddTestKeys(c.Ctx.DB, c.msmts[idx], tk)
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
