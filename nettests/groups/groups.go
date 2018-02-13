package groups

import (
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
