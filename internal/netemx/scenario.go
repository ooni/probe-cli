package netemx

import "github.com/ooni/netem"

const (
	// ScenarioRoleDNSOverHTTPS means we should create a DNS-over-HTTPS server.
	ScenarioRoleDNSOverHTTPS = iota

	// ScenarioRoleExampleLikeWebServer means we should instantiate a www.example.com-like web server.
	ScenarioRoleExampleLikeWebServer

	// ScenarioRoleOONIAPI means we should instantiate the OONI API.
	ScenarioRoleOONIAPI

	// ScenarioRoleUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioRoleUbuntuGeoIP

	// ScenarioRoleOONITestHelper means we should instantiate the oohelperd.
	ScenarioRoleOONITestHelper
)

// ScenarioDomainAddresses describes a domain and address used in a scenario.
type ScenarioDomainAddresses struct {
	Domain    string
	Addresses []string
	Role      uint64
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
	Role:      ScenarioRoleOONIAPI,
}, {
	Domain:    "geoip.ubuntu.com",
	Addresses: []string{"130.192.91.6"},
	Role:      ScenarioRoleUbuntuGeoIP,
}, {
	Domain:    "www.example.com",
	Addresses: []string{"130.192.91.7"},
	Role:      ScenarioRoleExampleLikeWebServer,
}, {
	Domain:    "0.th.ooni.org",
	Addresses: []string{"130.192.91.8"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domain:    "1.th.ooni.org",
	Addresses: []string{"130.192.91.9"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domain:    "2.th.ooni.org",
	Addresses: []string{"130.192.91.10"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domain:    "3.th.ooni.org",
	Addresses: []string{"130.192.91.11"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domain:    "dns.quad9.net",
	Addresses: []string{"130.192.91.12"},
	Role:      ScenarioRoleDNSOverHTTPS,
}, {
	Domain:    "mozilla.cloudflare-dns.com",
	Addresses: []string{"130.192.91.13"},
	Role:      ScenarioRoleDNSOverHTTPS,
}, {
	Domain:    "dns.google",
	Addresses: []string{"130.192.91.14"},
	Role:      ScenarioRoleDNSOverHTTPS,
}}

// MustNewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func MustNewScenario(cfg []*ScenarioDomainAddresses) *QAEnv {
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
		switch sad.Role {
		case ScenarioRoleDNSOverHTTPS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &DNSOverHTTPSHandlerFactory{
					Config: dohConfig,
				}))
			}

		case ScenarioRoleExampleLikeWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, ExampleWebPageHandlerFactory()))
			}

		case ScenarioRoleOONIAPI:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOAPIHandlerFactory{}))
			}

		case ScenarioRoleOONITestHelper:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOHelperDFactory{}))
			}

		case ScenarioRoleUbuntuGeoIP:
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
