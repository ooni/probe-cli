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
	ctx *ooni.Context
	res *database.Result
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) {
	log.Debugf("Init: %v", nt)
	nt.Options = mk.NettestOptions{
		IncludeIP:        c.ctx.Config.Sharing.IncludeIP,
		IncludeASN:       c.ctx.Config.Sharing.IncludeASN,
		IncludeCountry:   c.ctx.Config.Advanced.IncludeCountry,
		DisableCollector: false,
		SoftwareName:     "ooniprobe",
		SoftwareVersion:  version.Version,

		// XXX
		GeoIPCountryPath: "",
		GeoASNPath:       "",
		OutputPath:       "",
		CaBundlePath:     "",
	}
	nt.RegisterEventHandler(func(event interface{}) {
		e := event.(map[string]interface{})
		if e["type"].(string) == "LOG" {
			msg := e["message"].(string)
			switch level := e["verbosity"].(string); level {
			case "ERROR":
				log.Error(msg)
			case "INFO":
				log.Info(msg)
			default:
				log.Debug(msg)
			}
		} else {
			log.WithFields(log.Fields{
				"key":   "event",
				"value": e,
			}).Info("got event")
		}
	})
}

// OnProgress should be called when a new progress event is available.
func (c *Controller) OnProgress(perc float32, msg string) {
	log.Debugf("OnProgress: %f - %s", perc, msg)
}

// OnEntry should be called every time there is a new entry
func (c *Controller) OnEntry(entry string) {
	log.Debugf("OnEntry: %s", entry)
}

// MKStart is the interface for the mk.Nettest Start() function
type MKStart func(name string) (chan bool, error)

// Start should be called to start the test
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
