package nettests

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/measurement-kit/go-measurement-kit"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/internal/cli/version"
	"github.com/openobservatory/gooni/internal/colors"
	"github.com/openobservatory/gooni/internal/database"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	Summary(map[string]interface{}) interface{}
	LogSummary(string) error
}

// NewController creates a nettest controller
func NewController(nt Nettest, ctx *ooni.Context, res *database.Result, msmtPath string) *Controller {
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
	Ctx      *ooni.Context
	res      *database.Result
	nt       Nettest
	msmts    map[int64]*database.Measurement
	msmtPath string // XXX maybe we can drop this and just use a temporary file
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) error {
	log.Debugf("Init: %v", nt)
	c.msmts = make(map[int64]*database.Measurement)

	msmtTemplate := database.Measurement{
		ASN:            "",
		IP:             "",
		CountryCode:    "",
		ReportID:       "",
		Name:           nt.Name,
		ResultID:       c.res.ID,
		ReportFilePath: c.msmtPath,
	}

	log.Debugf("OutputPath: %s", c.msmtPath)
	nt.Options = mk.NettestOptions{
		IncludeIP:        c.Ctx.Config.Sharing.IncludeIP,
		IncludeASN:       c.Ctx.Config.Sharing.IncludeASN,
		IncludeCountry:   c.Ctx.Config.Advanced.IncludeCountry,
		DisableCollector: false,
		SoftwareName:     "ooniprobe",
		SoftwareVersion:  version.Version,

		// XXX
		GeoIPCountryPath: "",
		GeoIPASNPath:     "",
		OutputPath:       c.msmtPath,
		CaBundlePath:     "/etc/ssl/cert.pem",
	}

	nt.On("log", func(e mk.Event) {
		level := e.Value.LogLevel
		msg := e.Value.Message

		switch level {
		case "ERROR":
			log.Error(msg)
		case "INFO":
			log.Info(msg)
		default:
			log.Debug(msg)
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

		msmtTemplate.ReportID = e.Value.ReportID
	})

	nt.On("status.geoip_lookup", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		msmtTemplate.ASN = e.Value.ProbeASN
		msmtTemplate.IP = e.Value.ProbeIP
		msmtTemplate.CountryCode = e.Value.ProbeCC
	})

	nt.On("status.measurement_started", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		idx := e.Value.Idx
		msmt, err := database.CreateMeasurement(c.Ctx.DB, msmtTemplate, e.Value.Input)
		if err != nil {
			log.WithError(err).Error("Failed to create measurement")
			return
		}
		c.msmts[idx] = msmt
	})

	nt.On("status.progress", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))
		c.OnProgress(e.Value.Percentage, e.Value.Message)
	})

	nt.On("status.update.*", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))
	})

	nt.On("failure.measurement", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		c.msmts[e.Value.Idx].Failed(c.Ctx.DB, e.Value.Failure)
	})

	nt.On("failure.measurement_submission", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		failure := e.Value.Failure
		c.msmts[e.Value.Idx].UploadFailed(c.Ctx.DB, failure)
	})

	nt.On("status.measurement_uploaded", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		if err := c.msmts[e.Value.Idx].UploadSucceeded(c.Ctx.DB); err != nil {
			log.WithError(err).Error("failed to mark msmt as uploaded")
		}
	})

	nt.On("status.measurement_done", func(e mk.Event) {
		log.Debugf(colors.Red(e.Key))

		if err := c.msmts[e.Value.Idx].Done(c.Ctx.DB); err != nil {
			log.WithError(err).Error("failed to mark msmt as done")
		}
	})

	nt.On("measurement", func(e mk.Event) {
		c.OnEntry(e.Value.Idx, e.Value.JSONStr)
	})

	nt.On("status.end", func(e mk.Event) {
		log.Debugf("status.end")
		for idx, msmt := range c.msmts {
			log.Debugf("adding msmt#%d to result", idx)
			if err := msmt.AddToResult(c.Ctx.DB, c.res); err != nil {
				log.WithError(err).Error("failed to add to result")
			}
		}
	})

	return nil
}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	log.Debugf("OnProgress: %f - %s", perc, msg)
}

// Entry is an opaque measurement entry
type Entry struct {
	TestKeys map[string]interface{} `json:"test_keys"`
}

// OnEntry should be called every time there is a new entry
func (c *Controller) OnEntry(idx int64, jsonStr string) {
	log.Debugf("OnEntry")

	var entry Entry
	json.Unmarshal([]byte(jsonStr), &entry)
	summary := c.nt.Summary(entry.TestKeys)
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		log.WithError(err).Error("failed to serialize summary")
	}
	log.Debugf("Fetching: %s %v", idx, c.msmts[idx])
	c.msmts[idx].WriteSummary(c.Ctx.DB, string(summaryBytes))
}

// MKStart is the interface for the mk.Nettest Start() function
type MKStart func(name string) (chan bool, error)

// Start should be called to start the test
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
