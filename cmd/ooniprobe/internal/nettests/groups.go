package nettests

// Group is a group of nettests
type Group struct {
	Label        string
	Nettests     []Nettest
	UnattendedOK bool
}

// All contains all the nettests that can be run by the user
var All = map[string]Group{
	"websites": {
		Label: "Websites",
		Nettests: []Nettest{
			WebConnectivity{},
		},
		UnattendedOK: true,
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
		UnattendedOK: true,
	},
	"im": {
		Label: "Instant Messaging",
		Nettests: []Nettest{
			FacebookMessenger{},
			Telegram{},
			WhatsApp{},
			Signal{},
		},
		UnattendedOK: true,
	},
	"circumvention": {
		Label: "Circumvention Tools",
		Nettests: []Nettest{
			Psiphon{},
			RiseupVPN{},
			Tor{},
		},
		UnattendedOK: true,
	},
	"experimental": {
		Label: "Experimental Nettests",
		Nettests: []Nettest{
			DNSCheck{},
			STUNReachability{},
			TorSf{},
		},
	},
}
