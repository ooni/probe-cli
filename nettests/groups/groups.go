package groups

import (
	"github.com/openobservatory/gooni/nettests"
	"github.com/openobservatory/gooni/nettests/performance"
	"github.com/openobservatory/gooni/nettests/websites"
)

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []nettests.Nettest
	Summary  func(s string) string
}

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": NettestGroup{
		Label: "Websites",
		Nettests: []nettests.Nettest{
			websites.WebConnectivity{},
		},
		Summary: func(s string) string {
			return "{}"
		},
	},
	"performance": NettestGroup{
		Label: "Performance",
		Nettests: []nettests.Nettest{
			performance.Dash{},
			performance.NDT{},
		},
		Summary: func(s string) string {
			return "{}"
		},
	},
	"middleboxes": NettestGroup{
		Label:    "Middleboxes",
		Nettests: []nettests.Nettest{},
		Summary: func(s string) string {
			return "{}"
		},
	},
	"im": NettestGroup{
		Label:    "Instant Messaging",
		Nettests: []nettests.Nettest{},
		Summary: func(s string) string {
			return "{}"
		},
	},
}
