package nettests

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []Nettest
}

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": NettestGroup{
		Label: "Websites",
		Nettests: []Nettest{
			WebConnectivity{},
		},
	},
	"performance": NettestGroup{
		Label: "Performance",
		Nettests: []Nettest{
			Dash{},
			NDT{},
		},
	},
	"middlebox": NettestGroup{
		Label: "Middleboxes",
		Nettests: []Nettest{
			HTTPInvalidRequestLine{},
			HTTPHeaderFieldManipulation{},
		},
	},
	"im": NettestGroup{
		Label: "Instant Messaging",
		Nettests: []Nettest{
			FacebookMessenger{},
			Telegram{},
			WhatsApp{},
		},
	},
	"circumvention": NettestGroup{
		Label: "Circumvention Tools",
		Nettests: []Nettest{
			Psiphon{},
		},
	},
}
