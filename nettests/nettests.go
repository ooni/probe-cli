package nettests

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/measurement-kit/go-measurement-kit"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/internal/cli/version"
	"github.com/openobservatory/gooni/internal/database"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	Summary(map[string]interface{}) interface{}
	LogSummary(string) error
}

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []Nettest
	Summary  func(s string) string
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
	msmtPath string
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) error {
	log.Debugf("Init: %v", nt)

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
		level := e.Value["verbosity"].(string)
		msg := e.Value["message"].(string)

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

		msmtTemplate.ReportID = e.Value["report_id"].(string)
	})

	nt.On("status.geoip_lookup", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		msmtTemplate.ASN = e.Value["probe_asn"].(string)
		msmtTemplate.IP = e.Value["probe_ip"].(string)
		msmtTemplate.CountryCode = e.Value["probe_cc"].(string)
	})

	nt.On("status.measurement_started", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		idx := e.Value["idx"].(int64)
		input := e.Value["input"].(string)
		msmt, err := database.CreateMeasurement(c.Ctx.DB, msmtTemplate, input)
		if err != nil {
			log.WithError(err).Error("Failed to create measurement")
			return
		}
		c.msmts[idx] = msmt
	})

	nt.On("status.progress", func(e mk.Event) {
		perc := e.Value["percentage"].(float64)
		msg := e.Value["message"].(string)
		c.OnProgress(perc, msg)
	})

	nt.On("status.update.*", func(e mk.Event) {
		log.Debugf("%s", e.Key)
	})

	nt.On("failure.measurement", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		idx := e.Value["idx"].(int64)
		failure := e.Value["failure"].(string)
		c.msmts[idx].Failed(c.Ctx.DB, failure)
	})

	nt.On("failure.measurement_submission", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		idx := e.Value["idx"].(int64)
		failure := e.Value["failure"].(string)
		c.msmts[idx].UploadFailed(c.Ctx.DB, failure)
	})

	nt.On("status.measurement_uploaded", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		idx := e.Value["idx"].(int64)
		c.msmts[idx].UploadSucceeded(c.Ctx.DB)
	})

	nt.On("status.measurement_done", func(e mk.Event) {
		log.Debugf("%s", e.Key)

		idx := e.Value["idx"].(int64)
		c.msmts[idx].Done(c.Ctx.DB)
	})

	nt.On("measurement", func(e mk.Event) {
		idx := e.Value["idx"].(int64)
		c.OnEntry(idx, e.Value["json_str"].(string))
	})

	nt.On("end", func(e mk.Event) {
		log.Debugf("end")
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
	log.Debugf("OnEntry: %s", jsonStr)

	var entry Entry
	json.Unmarshal([]byte(jsonStr), &entry)
	summary := c.nt.Summary(entry.TestKeys)
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		log.WithError(err).Error("failed to serialize summary")
	}
	c.msmts[idx].WriteSummary(c.Ctx.DB, string(summaryBytes))
}

// MKStart is the interface for the mk.Nettest Start() function
type MKStart func(name string) (chan bool, error)

// Start should be called to start the test
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
