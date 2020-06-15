package nettests

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []Nettest
}

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": {
		Label: "Websites",
		Nettests: []Nettest{
			WebConnectivity{},
		},
	},
	"performance": {
		Label: "Performance",
		Nettests: []Nettest{
			Dash{},
			NDT{},
		},
	},
	"middlebox": {
		Label: "Middleboxes",
		Nettests: []Nettest{
			HTTPInvalidRequestLine{},
			HTTPHeaderFieldManipulation{},
		},
	},
	"im": {
		Label: "Instant Messaging",
		Nettests: []Nettest{
			FacebookMessenger{},
			Telegram{},
			WhatsApp{},
		},
	},
	"circumvention": {
		Label: "Circumvention Tools",
		Nettests: []Nettest{
			STUNReachability{},
			Psiphon{},
			Tor{},
		},
	},
}
