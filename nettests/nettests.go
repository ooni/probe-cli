package nettests

import (
	"github.com/apex/log"
	"github.com/measurement-kit/go-measurement-kit"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/internal/database"
)

// Nettest interface. Every Nettest should implement this.
type Nettest interface {
	Run(*Controller) error
	Summary(*database.Measurement) string
	LogSummary(string) error
}

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []Nettest
	Summary  func(s string) string
}

// Controller is passed to the run method of every Nettest
type Controller struct {
	ctx *ooni.Context
}

// New Nettest Controller
func (c *Controller) New(ctx *ooni.Context) *Controller {
	return &Controller{
		ctx,
	}
}

// Init should be called once to initialise the nettest
func (c *Controller) Init(nt *mk.Nettest) {
	log.Debugf("Init: %s", nt)
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
