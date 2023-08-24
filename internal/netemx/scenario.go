package netemx

import "github.com/ooni/netem"

const (
	// ScenarioFlagDNSOverHTTPS means we should create a DNS-over-HTTPS server.
	ScenarioFlagDNSOverHTTPS = 1 << iota

	// ScenarioFlagExampleLikeWebServer means we should instantiate a www.example.com-like web server.
	ScenarioFlagExampleLikeWebServer

	// ScenarioFlagOONIAPI means we should instantiate the OONI API.
	ScenarioFlagOONIAPI

	// ScenarioFlagUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioFlagUbuntuGeoIP

	// ScenarioFlagOONITestHelper means we should instantiate the oohelperd.
	ScenarioFlagOONITestHelper
)

// ScenarioDomainAddresses describes a domain and address used in a scenario.
type ScenarioDomainAddresses struct {
	Domain    string
	Addresses []string
	Flags     uint64
}

// InternetScenario contains the domains and addresses used by [NewInternetScenario].
//
// Note that the 130.192.91.x address space belongs to polito.it and is not used for hosting
// servers, therefore we're more confident that tests using this scenario will break in bad
// way if for some reason netem is not working as intended. (We have several tests making sure
// of that, but some extra robustness won't hurt.)
var InternetScenario = []*ScenarioDomainAddresses{{
	Domain:    "api.ooni.io",
	Addresses: []string{"130.192.91.5"},
	Flags:     ScenarioFlagOONIAPI,
}, {
	Domain:    "geoip.ubuntu.com",
	Addresses: []string{"130.192.91.6"},
	Flags:     ScenarioFlagUbuntuGeoIP,
}, {
	Domain:    "www.example.com",
	Addresses: []string{"130.192.91.7"},
	Flags:     ScenarioFlagExampleLikeWebServer,
}, {
	Domain:    "0.th.ooni.org",
	Addresses: []string{"130.192.91.8"},
	Flags:     ScenarioFlagOONITestHelper,
}, {
	Domain:    "1.th.ooni.org",
	Addresses: []string{"130.192.91.9"},
	Flags:     ScenarioFlagOONITestHelper,
}, {
	Domain:    "2.th.ooni.org",
	Addresses: []string{"130.192.91.10"},
	Flags:     ScenarioFlagOONITestHelper,
}, {
	Domain:    "3.th.ooni.org",
	Addresses: []string{"130.192.91.11"},
	Flags:     ScenarioFlagOONITestHelper,
}, {
	Domain:    "dns.quad9.net",
	Addresses: []string{"130.192.91.12"},
	Flags:     ScenarioFlagDNSOverHTTPS,
}, {
	Domain:    "mozilla.cloudflare-dns.com",
	Addresses: []string{"130.192.91.13"},
	Flags:     ScenarioFlagDNSOverHTTPS,
}, {
	Domain:    "dns.google",
	Addresses: []string{"130.192.91.14"},
	Flags:     ScenarioFlagDNSOverHTTPS,
}}

// NewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func NewScenario(cfg []*ScenarioDomainAddresses) *QAEnv {
	var opts []QAEnvOption

	// create a common configuration for DoH servers
	dohConfig := netem.NewDNSConfig()
	for _, sad := range cfg {
		dohConfig.AddRecord(sad.Domain, "", sad.Addresses...)
	}

	// explicitly create the uncensored resolver
	opts = append(opts, QAEnvOptionDNSOverUDPResolvers("130.192.91.4"))

	// fill options based on the scenario config
	for _, sad := range cfg {
		if (sad.Flags & ScenarioFlagDNSOverHTTPS) != 0 {
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &DNSOverHTTPSHandlerFactory{
					Config: dohConfig,
				}))
			}
		}

		if (sad.Flags & ScenarioFlagExampleLikeWebServer) != 0 {
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, ExampleWebPageHandlerFactory()))
			}
		}

		if (sad.Flags & ScenarioFlagOONIAPI) != 0 {
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOAPIHandlerFactory{}))
			}
		}

		if (sad.Flags & ScenarioFlagOONITestHelper) != 0 {
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOHelperDFactory{}))
			}
		}

		if (sad.Flags & ScenarioFlagUbuntuGeoIP) != 0 {
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &GeoIPHandlerFactoryUbuntu{
					ProbeIP: QAEnvDefaultClientAddress,
				}))
			}
		}
	}

	// create the QAEnv
	env := MustNewQAEnv(opts...)

	// configure all the domain names
	for _, sad := range cfg {
		env.AddRecordToAllResolvers(sad.Domain, "", sad.Addresses...)
	}

	return env
}
