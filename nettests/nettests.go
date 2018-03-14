package nettests

import (
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
func NewController(ctx *ooni.Context, res *database.Result) *Controller {
	return &Controller{
		ctx,
		res,
	}
}

// Controller is passed to the run method of every Nettest
type Controller struct {
	Ctx *ooni.Context
	res *database.Result
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) {
	log.Debugf("Init: %v", nt)
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
		OutputPath:       "/tmp/measurement.jsonl",
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
	})

	nt.On("status.geoip_lookup", func(e mk.Event) {
		log.Debugf("%s", e.Key)
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
	})

	nt.On("failure.report_submission", func(e mk.Event) {
		log.Debugf("%s", e.Key)
	})

	nt.On("measurement", func(e mk.Event) {
		c.OnEntry(e.Value["json_str"].(string))
	})

}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float64, msg string) {
	log.Debugf("OnProgress: %f - %s", perc, msg)
}

// OnEntry should be called every time there is a new entry
func (c *Controller) OnEntry(jsonStr string) {
	log.Debugf("OnEntry: %s", jsonStr)
}

// MKStart is the interface for the mk.Nettest Start() function
type MKStart func(name string) (chan bool, error)

// Start should be called to start the test
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
