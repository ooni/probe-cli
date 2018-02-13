package groups

import (
	"github.com/apex/log"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/nettests"
	"github.com/openobservatory/gooni/nettests/performance"
	"github.com/openobservatory/gooni/nettests/websites"
)

// NettestGroups that can be run by the user
var NettestGroups = map[string]nettests.NettestGroup{
	"websites": nettests.NettestGroup{
		Label: "Websites",
		Nettests: []nettests.Nettest{
			websites.WebConnectivity{},
		},
	},
	"performance": nettests.NettestGroup{
		Label: "Performance",
		Nettests: []nettests.Nettest{
			performance.NDT{},
		},
	},
	"middleboxes": nettests.NettestGroup{
		Label:    "Middleboxes",
		Nettests: []nettests.Nettest{},
	},
	"im": nettests.NettestGroup{
		Label:    "Instant Messaging",
		Nettests: []nettests.Nettest{},
	},
}

// Run runs a specific test group
func Run(name string, ctx *ooni.Context) error {
	group := NettestGroups[name]
	log.Debugf("Running test group %s", group.Label)

	for _, nt := range group.Nettests {
		ctl := nettests.NewController(ctx)
		nt.Run(ctl)
	}
	return nil
}
