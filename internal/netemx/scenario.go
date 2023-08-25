package netemx

import "github.com/ooni/netem"

const (
	// ScenarioRoleDNSOverHTTPS means we should create a DNS-over-HTTPS server.
	ScenarioRoleDNSOverHTTPS = iota

	// ScenarioRoleWebServer means we should instantiate a webserver using a specific factory.
	ScenarioRoleWebServer

	// ScenarioRoleOONIAPI means we should instantiate the OONI API.
	ScenarioRoleOONIAPI

	// ScenarioRoleUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioRoleUbuntuGeoIP

	// ScenarioRoleOONITestHelper means we should instantiate the oohelperd.
	ScenarioRoleOONITestHelper
)

// ScenarioDomainAddresses describes a domain and address used in a scenario.
type ScenarioDomainAddresses struct {
	// Domains contains a related set of domains domains (MANDATORY field).
	Domains []string

	// Addresses contains the MANDATORY list of addresses belonging to the domain.
	Addresses []string

	// Role is the MANDATORY role of this domain.
	Role uint64

	// WebServerFactory is the factory to use when Role is ScenarioRoleWebServer.
	WebServerFactory QAEnvHTTPHandlerFactory
}

// InternetScenario contains the domains and addresses used by [NewInternetScenario].
//
// Note that the 130.192.91.x address space belongs to polito.it and is not used for hosting
// servers, therefore we're more confident that tests using this scenario will break in bad
// way if for some reason netem is not working as intended. (We have several tests making sure
// of that, but some extra robustness won't hurt.)
var InternetScenario = []*ScenarioDomainAddresses{{
	Domains:   []string{"api.ooni.io"},
	Addresses: []string{"130.192.91.5"},
	Role:      ScenarioRoleOONIAPI,
}, {
	Domains:   []string{"geoip.ubuntu.com"},
	Addresses: []string{"130.192.91.6"},
	Role:      ScenarioRoleUbuntuGeoIP,
}, {
	Domains:          []string{"www.example.com", "example.com"},
	Addresses:        []string{"130.192.91.7"},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: ExampleWebPageHandlerFactory(),
}, {
	Domains:   []string{"0.th.ooni.org"},
	Addresses: []string{"130.192.91.8"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domains:   []string{"1.th.ooni.org"},
	Addresses: []string{"130.192.91.9"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domains:   []string{"2.th.ooni.org"},
	Addresses: []string{"130.192.91.10"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domains:   []string{"3.th.ooni.org"},
	Addresses: []string{"130.192.91.11"},
	Role:      ScenarioRoleOONITestHelper,
}, {
	Domains:   []string{"dns.quad9.net"},
	Addresses: []string{"130.192.91.12"},
	Role:      ScenarioRoleDNSOverHTTPS,
}, {
	Domains:   []string{"mozilla.cloudflare-dns.com"},
	Addresses: []string{"130.192.91.13"},
	Role:      ScenarioRoleDNSOverHTTPS,
}, {
	Domains:   []string{"dns.google"},
	Addresses: []string{"130.192.91.14"},
	Role:      ScenarioRoleDNSOverHTTPS,
}, {
	Domains:          nil,
	Addresses:        []string{"130.192.91.15"},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: BlockpageHandlerFactory(),
}, {
	Domains:          []string{"www.example.org", "example.org"},
	Addresses:        []string{"130.192.91.16"},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: ExampleWebPageHandlerFactory(),
}}

// MustNewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func MustNewScenario(config []*ScenarioDomainAddresses) *QAEnv {
	var opts []QAEnvOption

	// create a common configuration for DoH servers
	dohConfig := netem.NewDNSConfig()
	for _, sad := range config {
		for _, domain := range sad.Domains {
			dohConfig.AddRecord(domain, "", sad.Addresses...)
		}
	}

	// explicitly create the uncensored resolver
	opts = append(opts, QAEnvOptionDNSOverUDPResolvers("130.192.91.4"))

	// fill options based on the scenario config
	for _, sad := range config {
		switch sad.Role {
		case ScenarioRoleDNSOverHTTPS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &DNSOverHTTPSHandlerFactory{
					Config: dohConfig,
				}))
			}

		case ScenarioRoleWebServer:
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
	for _, sad := range config {
		for _, domain := range sad.Domains {
			env.AddRecordToAllResolvers(domain, "", sad.Addresses...)
		}
	}

	return env
}
