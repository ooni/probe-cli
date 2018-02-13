package nettests

import (
	"fmt"

	"github.com/apex/log"
	"github.com/measurement-kit/go-measurement-kit"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/internal/cli/version"
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
func NewController(ctx *ooni.Context) *Controller {
	return &Controller{
		ctx,
	}
}

// Controller is passed to the run method of every Nettest
type Controller struct {
	ctx *ooni.Context
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) {
	log.Debugf("Init: %s", nt)
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
		fmt.Println("Got event", event)
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

// Start should be called every time there is a new entry
func (c *Controller) Start(f MKStart) {
	log.Debugf("MKStart: %s", f)
}
