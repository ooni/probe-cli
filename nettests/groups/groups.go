package groups

import (
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-cli/nettests/im"
	"github.com/ooni/probe-cli/nettests/middlebox"
	"github.com/ooni/probe-cli/nettests/performance"
	"github.com/ooni/probe-cli/nettests/websites"
)

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []nettests.Nettest
}

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": NettestGroup{
		Label: "Websites",
		Nettests: []nettests.Nettest{
			websites.WebConnectivity{},
		},
	},
	"performance": NettestGroup{
		Label: "Performance",
		Nettests: []nettests.Nettest{
			performance.Dash{},
			performance.NDT{},
		},
	},
	"middlebox": NettestGroup{
		Label: "Middleboxes",
		Nettests: []nettests.Nettest{
			middlebox.HTTPInvalidRequestLine{},
			middlebox.HTTPHeaderFieldManipulation{},
		},
	},
	"im": NettestGroup{
		Label: "Instant Messaging",
		Nettests: []nettests.Nettest{
			im.FacebookMessenger{},
			im.Telegram{},
			im.WhatsApp{},
		},
	},
}
